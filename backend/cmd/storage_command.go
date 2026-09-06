package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	local_storage "databasus-backend/internal/features/storages/models/local"
	"databasus-backend/internal/util/encryption"
)

const (
	storagePermissionsDocumentationURL = "https://databasus.com/advanced-config/#docker-storage-permissions"
	storageProbeContent                = "databasus storage test"
)

type storageFileSaverAndDeleter interface {
	SaveFile(
		ctx context.Context,
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		fileName string,
		file io.Reader,
	) error
	DeleteFile(
		ctx context.Context,
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		fileName string,
	) error
}

type storageTestCommandWriters struct {
	standardOutput io.Writer
	errorOutput    io.Writer
}

type storageTestCommand struct {
	storage         storageFileSaverAndDeleter
	fieldEncryptor  encryption.FieldEncryptor
	logger          *slog.Logger
	writers         storageTestCommandWriters
	temporaryFolder string
	backupFolder    string
}

func newStorageTestCommand(
	logger *slog.Logger,
	writers storageTestCommandWriters,
) storageTestCommand {
	environment := config.GetEnv()

	return storageTestCommand{
		storage:         &local_storage.LocalStorage{},
		fieldEncryptor:  encryption.GetFieldEncryptor(),
		logger:          logger,
		writers:         writers,
		temporaryFolder: environment.TempFolder,
		backupFolder:    environment.DataFolder,
	}
}

func (c storageTestCommand) SaveAndDeleteProbe(ctx context.Context) int {
	probeFileName := ".databasus-storage-test-" + uuid.NewString()
	probeContent := strings.NewReader(storageProbeContent)

	if err := c.storage.SaveFile(ctx, c.fieldEncryptor, c.logger, probeFileName, probeContent); err != nil {
		c.removeTemporaryProbeAfterFailedSave(ctx, probeFileName)
		c.reportFailure(ctx, "save a file through local storage", err)
		return 1
	}

	if err := c.storage.DeleteFile(ctx, c.fieldEncryptor, c.logger, probeFileName); err != nil {
		c.reportFailure(ctx, "delete a file through local storage", err)
		return 1
	}

	if _, err := fmt.Fprintln(c.writers.standardOutput, "storage test passed"); err != nil {
		return 1
	}

	return 0
}

func (c storageTestCommand) removeTemporaryProbeAfterFailedSave(ctx context.Context, probeFileName string) {
	temporaryProbePath := filepath.Join(c.temporaryFolder, probeFileName)
	if err := os.Remove(temporaryProbePath); err != nil && !os.IsNotExist(err) {
		c.logger.WarnContext(
			ctx,
			"failed to remove storage test temporary file",
			"path",
			temporaryProbePath,
			"error",
			err,
		)
	}
}

func (c storageTestCommand) reportFailure(
	ctx context.Context,
	operation string,
	operationError error,
) {
	_, writeError := fmt.Fprintf(
		c.writers.errorOutput,
		"ERROR: Databasus cannot write to local storage paths %s and %s as UID %d and GID %d.\n"+
			"Required operation: %s.\n"+
			"Set PUID and PGID or fix the mounted directory permissions: %s\n"+
			"Details: %v\n",
		c.temporaryFolder,
		c.backupFolder,
		os.Geteuid(),
		os.Getegid(),
		operation,
		storagePermissionsDocumentationURL,
		operationError,
	)
	if writeError != nil {
		c.logger.ErrorContext(ctx, "failed to write storage test error", "error", writeError)
	}
}
