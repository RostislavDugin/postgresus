package local_storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	"databasus-backend/internal/util/encryption"
	files_utils "databasus-backend/internal/util/files"
)

const (
	// Chunk size for local storage writes - 8MB per buffer with double-buffering
	// allows overlapped I/O while keeping total memory under 32MB.
	// Two 8MB buffers = 16MB for local storage, plus 8MB for pg_dump buffer = ~25MB total.
	localChunkSize = 8 * 1024 * 1024
)

// LocalStorage uses ./databasus_local_backups folder as a
// directory for backups and ./databasus_local_temp folder as a
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
	fileName string,
	file io.Reader,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.InfoContext(ctx, "starting to save file to local storage", "file_name", fileName)

	tempFilePath := filepath.Join(config.GetEnv().TempFolder, fileName)

	err := files_utils.EnsureDirectories([]string{
		config.GetEnv().TempFolder,
		filepath.Dir(tempFilePath),
	})
	if err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}
	logger.DebugContext(ctx, "creating temp file", "file_name", fileName, "temp_path", tempFilePath)

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to create temp file",
			"file_name",
			fileName,
			"temp_path",
			tempFilePath,
			"error",
			err,
		)
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = tempFile.Close()
	}()

	logger.DebugContext(ctx, "copying file data to temp file", "file_name", fileName)
	_, err = copyWithContext(ctx, tempFile, file)
	if err != nil {
		logger.ErrorContext(ctx, "failed to write to temp file", "file_name", fileName, "error", err)
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err = tempFile.Sync(); err != nil {
		logger.ErrorContext(ctx, "failed to sync temp file", "file_name", fileName, "error", err)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close the temp file explicitly before moving it (required on Windows)
	if err = tempFile.Close(); err != nil {
		logger.ErrorContext(ctx, "failed to close temp file", "file_name", fileName, "error", err)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	finalPath := filepath.Join(config.GetEnv().DataFolder, fileName)
	logger.DebugContext(
		ctx,
		"moving file from temp to final location",
		"file_name",
		fileName,
		"final_path",
		finalPath,
	)

	if err = files_utils.EnsureDirectories([]string{filepath.Dir(finalPath)}); err != nil {
		return fmt.Errorf("failed to ensure final directory: %w", err)
	}

	// Move the file from temp to backups directory
	if err = moveFile(tempFilePath, finalPath); err != nil {
		logger.ErrorContext(
			ctx,
			"failed to move file from temp to backups",
			"file_name",
			fileName,
			"temp_path",
			tempFilePath,
			"final_path",
			finalPath,
			"error",
			err,
		)
		return fmt.Errorf("failed to move file from temp to backups: %w", err)
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to local storage",
		"file_name",
		fileName,
		"final_path",
		finalPath,
	)

	return nil
}

func (l *LocalStorage) GetFile(
	_ context.Context,
	encryptor encryption.FieldEncryptor,
	_ *slog.Logger,
	fileName string,
) (io.ReadCloser, error) {
	filePath := filepath.Join(config.GetEnv().DataFolder, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", fileName)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

func (l *LocalStorage) DeleteFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) error {
	filePath := filepath.Join(config.GetEnv().DataFolder, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from local storage", "file_name", fileName)

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

// moveFile moves a file from src to dst. It first attempts os.Rename for efficiency.
// If rename fails with a cross-device link error (EXDEV), it falls back to copy-then-delete.
// This happens when users mount temp and backups directories as separate Docker volumes
// (e.g., on Unraid with split volume mapping), causing them to reside on different filesystems.
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(linkErr.Err, syscall.EXDEV) {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		_ = srcFile.Close()
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		_ = dstFile.Close()
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err = dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	if err = os.Remove(src); err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
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
