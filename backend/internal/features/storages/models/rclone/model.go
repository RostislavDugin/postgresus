package rclone_storage

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/operations"

	"databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
)

const (
	rcloneOperationTimeout = 30 * time.Second
	rcloneDeleteTimeout    = 30 * time.Second
)

var rcloneConfigMu sync.Mutex

type RcloneStorage struct {
	StorageID     uuid.UUID `json:"storageId"     gorm:"primaryKey;type:uuid;column:storage_id"`
	ConfigContent string    `json:"configContent" gorm:"not null;type:text;column:config_content"`
	RemotePath    string    `json:"remotePath"    gorm:"type:text;column:remote_path"`
}

func (r *RcloneStorage) TableName() string {
	return "rclone_storages"
}

func (r *RcloneStorage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
	file io.Reader,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.Info("Starting to save file to rclone storage", "fileName", fileName)

	remoteFs, err := r.getFs(ctx, encryptor)
	if err != nil {
		logger.Error("Failed to create rclone filesystem", "fileName", fileName, "error", err)
		return fmt.Errorf("failed to create rclone filesystem: %w", err)
	}

	filePath := r.getFilePath(fileName)
	logger.Debug("Uploading file via rclone", "fileName", fileName, "filePath", filePath)

	_, err = operations.Rcat(ctx, remoteFs, filePath, io.NopCloser(file), time.Now().UTC(), nil)
	if err != nil {
		select {
		case <-ctx.Done():
			logger.Info("Rclone upload cancelled", "fileName", fileName)
			return ctx.Err()
		default:
			logger.Error(
				"Failed to upload file via rclone",
				"fileName",
				fileName,
				"error",
				err,
			)
			return fmt.Errorf("failed to upload file via rclone: %w", err)
		}
	}

	logger.Info(
		"Successfully saved file to rclone storage",
		"fileName",
		fileName,
		"filePath",
		filePath,
	)
	return nil
}

func (r *RcloneStorage) GetFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) (io.ReadCloser, error) {
	remoteFs, err := r.getFs(ctx, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create rclone filesystem: %w", err)
	}

	filePath := r.getFilePath(fileName)

	obj, err := remoteFs.NewObject(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get object from rclone: %w", err)
	}

	return io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: obj.Size(),
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			if offsetBytes == 0 {
				return obj.Open(attemptCtx)
			}

			return obj.Open(attemptCtx, &fs.RangeOption{Start: offsetBytes, End: -1})
		},
		IsRetryableError: isRetryableRcloneError,
	}), nil
}

func isRetryableRcloneError(err error) bool {
	return !errors.Is(err, fs.ErrorObjectNotFound) &&
		!errors.Is(err, fs.ErrorDirNotFound) &&
		!errors.Is(err, fs.ErrorNotAFile) &&
		!errors.Is(err, fs.ErrorPermissionDenied)
}

func (r *RcloneStorage) DeleteFile(encryptor encryption.FieldEncryptor, fileName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rcloneDeleteTimeout)
	defer cancel()

	remoteFs, err := r.getFs(ctx, encryptor)
	if err != nil {
		return fmt.Errorf("failed to create rclone filesystem: %w", err)
	}

	filePath := r.getFilePath(fileName)

	obj, err := remoteFs.NewObject(ctx, filePath)
	if errors.Is(err, fs.ErrorObjectNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to look up file in rclone: %w", err)
	}

	if err := operations.DeleteFile(ctx, obj); err != nil {
		return fmt.Errorf("failed to delete file via rclone: %w", err)
	}

	return nil
}

func (r *RcloneStorage) Validate(encryptor encryption.FieldEncryptor) error {
	if r.ConfigContent == "" {
		return errors.New("rclone config content is required")
	}

	configContent, err := encryptor.Decrypt(r.ConfigContent)
	if err != nil {
		return fmt.Errorf("failed to decrypt rclone config content: %w", err)
	}

	parsedConfig, err := parseConfigContent(configContent)
	if err != nil {
		return fmt.Errorf("failed to parse rclone config: %w", err)
	}

	if len(parsedConfig) == 0 {
		return errors.New("rclone config must contain at least one remote section")
	}

	if len(parsedConfig) > 1 {
		return fmt.Errorf(
			"rclone config must contain exactly one remote section, but found %d; create a separate storage for each remote",
			len(parsedConfig),
		)
	}

	return nil
}

func (r *RcloneStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	ctx, cancel := context.WithTimeout(context.Background(), rcloneOperationTimeout)
	defer cancel()

	remoteFs, err := r.getFs(ctx, encryptor)
	if err != nil {
		return fmt.Errorf("failed to create rclone filesystem: %w", err)
	}

	testFileID := uuid.New().String() + "-test"
	testFilePath := r.getFilePath(testFileID)
	testData := strings.NewReader("test connection")

	_, err = operations.Rcat(
		ctx,
		remoteFs,
		testFilePath,
		io.NopCloser(testData),
		time.Now().UTC(),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to upload test file via rclone: %w", err)
	}

	obj, err := remoteFs.NewObject(ctx, testFilePath)
	if err != nil {
		return fmt.Errorf("failed to get test file from rclone: %w", err)
	}

	err = obj.Remove(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete test file from rclone: %w", err)
	}

	return nil
}

func (r *RcloneStorage) HideSensitiveData() {
	r.ConfigContent = ""
}

func (r *RcloneStorage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if r.ConfigContent != "" {
		encrypted, err := encryptor.Encrypt(r.ConfigContent)
		if err != nil {
			return fmt.Errorf("failed to encrypt rclone config content: %w", err)
		}
		r.ConfigContent = encrypted
	}

	return nil
}

func (r *RcloneStorage) Update(incoming *RcloneStorage) {
	r.RemotePath = incoming.RemotePath

	if incoming.ConfigContent != "" {
		r.ConfigContent = incoming.ConfigContent
	}
}

func (r *RcloneStorage) getFs(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) (fs.Fs, error) {
	configContent, err := encryptor.Decrypt(r.ConfigContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt rclone config content: %w", err)
	}

	rcloneConfigMu.Lock()
	defer rcloneConfigMu.Unlock()

	parsedConfig, err := parseConfigContent(configContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rclone config: %w", err)
	}

	if len(parsedConfig) == 0 {
		return nil, errors.New("rclone config must contain at least one remote section")
	}

	if len(parsedConfig) > 1 {
		return nil, fmt.Errorf(
			"rclone config must contain exactly one remote section, but found %d; create a separate storage for each remote",
			len(parsedConfig),
		)
	}

	var remoteName string
	for section, values := range parsedConfig {
		remoteName = section
		for key, value := range values {
			config.FileSetValue(section, key, value)
		}
	}

	remotePath := remoteName + ":"
	if r.RemotePath != "" {
		remotePath = remoteName + ":" + strings.TrimPrefix(r.RemotePath, "/")
	}

	remoteFs, err := fs.NewFs(ctx, remotePath)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create rclone filesystem for remote '%s': %w",
			remoteName,
			err,
		)
	}

	return remoteFs, nil
}

func (r *RcloneStorage) getFilePath(filename string) string {
	return filename
}

func parseConfigContent(content string) (map[string]map[string]string, error) {
	sections := make(map[string]map[string]string)

	var currentSection string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimPrefix(strings.TrimSuffix(line, "]"), "[")
			if sections[currentSection] == nil {
				sections[currentSection] = make(map[string]string)
			}
			continue
		}

		if currentSection != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := ""
			if len(parts) > 1 {
				value = strings.TrimSpace(parts[1])
			}
			sections[currentSection][key] = value
		}
	}

	return sections, scanner.Err()
}
