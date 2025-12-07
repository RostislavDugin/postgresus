package local_storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"postgresus-backend/internal/config"
	"postgresus-backend/internal/util/encryption"
	files_utils "postgresus-backend/internal/util/files"

	"github.com/google/uuid"
)

const (
	// Chunk size for local storage writes - 8MB per buffer with double-buffering
	// allows overlapped I/O while keeping total memory under 32MB.
	// Two 8MB buffers = 16MB for local storage, plus 8MB for pg_dump buffer = ~25MB total.
	localChunkSize = 8 * 1024 * 1024
)

// LocalStorage uses ./postgresus_local_backups folder as a
// directory for backups and ./postgresus_local_temp folder as a
// directory for temp files
type LocalStorage struct {
	StorageID uuid.UUID `json:"storageId" gorm:"primaryKey;type:uuid;column:storage_id"`
}

func (l *LocalStorage) TableName() string {
	return "local_storages"
}

func (l *LocalStorage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileID uuid.UUID,
	file io.Reader,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.Info("Starting to save file to local storage", "fileId", fileID.String())

	err := files_utils.EnsureDirectories([]string{
		config.GetEnv().TempFolder,
	})
	if err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	tempFilePath := filepath.Join(config.GetEnv().TempFolder, fileID.String())
	logger.Debug("Creating temp file", "fileId", fileID.String(), "tempPath", tempFilePath)

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		logger.Error(
			"Failed to create temp file",
			"fileId",
			fileID.String(),
			"tempPath",
			tempFilePath,
			"error",
			err,
		)
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = tempFile.Close()
	}()

	logger.Debug("Copying file data to temp file", "fileId", fileID.String())
	_, err = copyWithContext(ctx, tempFile, file)
	if err != nil {
		logger.Error("Failed to write to temp file", "fileId", fileID.String(), "error", err)
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err = tempFile.Sync(); err != nil {
		logger.Error("Failed to sync temp file", "fileId", fileID.String(), "error", err)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close the temp file explicitly before moving it (required on Windows)
	if err = tempFile.Close(); err != nil {
		logger.Error("Failed to close temp file", "fileId", fileID.String(), "error", err)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	finalPath := filepath.Join(config.GetEnv().DataFolder, fileID.String())
	logger.Debug(
		"Moving file from temp to final location",
		"fileId",
		fileID.String(),
		"finalPath",
		finalPath,
	)

	// Move the file from temp to backups directory
	if err = os.Rename(tempFilePath, finalPath); err != nil {
		logger.Error(
			"Failed to move file from temp to backups",
			"fileId",
			fileID.String(),
			"tempPath",
			tempFilePath,
			"finalPath",
			finalPath,
			"error",
			err,
		)
		return fmt.Errorf("failed to move file from temp to backups: %w", err)
	}

	logger.Info(
		"Successfully saved file to local storage",
		"fileId",
		fileID.String(),
		"finalPath",
		finalPath,
	)

	return nil
}

func (l *LocalStorage) GetFile(
	encryptor encryption.FieldEncryptor,
	fileID uuid.UUID,
) (io.ReadCloser, error) {
	filePath := filepath.Join(config.GetEnv().DataFolder, fileID.String())

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", fileID.String())
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

func (l *LocalStorage) DeleteFile(encryptor encryption.FieldEncryptor, fileID uuid.UUID) error {
	filePath := filepath.Join(config.GetEnv().DataFolder, fileID.String())

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (l *LocalStorage) Validate(encryptor encryption.FieldEncryptor) error {
	return nil
}

func (l *LocalStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	testFile := filepath.Join(config.GetEnv().TempFolder, "test_connection")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("failed to close test file: %w", err)
	}

	if err = os.Remove(testFile); err != nil {
		return fmt.Errorf("failed to remove test file: %w", err)
	}

	return nil
}

func (l *LocalStorage) HideSensitiveData() {
}

func (l *LocalStorage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	return nil
}

func (l *LocalStorage) Update(incoming *LocalStorage) {
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, localChunkSize)
	var written int64

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := dst.Write(buf[:nr])
			written += int64(nw)
			if writeErr != nil {
				return written, writeErr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}

		if readErr == io.EOF {
			return written, nil
		}
		if readErr != nil {
			return written, readErr
		}
	}
}
