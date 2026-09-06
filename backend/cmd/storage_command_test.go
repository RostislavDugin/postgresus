package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/encryption"
)

type fakeStorageFileSaverAndDeleter struct {
	saveError       error
	deleteError     error
	savedFileName   string
	deletedFileName string
	savedContent    string
	temporaryFolder string
}

func (f *fakeStorageFileSaverAndDeleter) SaveFile(
	_ context.Context,
	_ encryption.FieldEncryptor,
	_ *slog.Logger,
	fileName string,
	file io.Reader,
) error {
	f.savedFileName = fileName

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	f.savedContent = string(fileContent)
	if f.temporaryFolder != "" {
		if err := os.WriteFile(
			filepath.Join(f.temporaryFolder, fileName),
			fileContent,
			0o600,
		); err != nil {
			return err
		}
	}

	return f.saveError
}

func (f *fakeStorageFileSaverAndDeleter) DeleteFile(
	_ context.Context,
	_ encryption.FieldEncryptor,
	_ *slog.Logger,
	fileName string,
) error {
	f.deletedFileName = fileName

	return f.deleteError
}

func Test_StorageTestCommand_SavesAndDeletesProbe(t *testing.T) {
	storage := &fakeStorageFileSaverAndDeleter{}
	output := &bytes.Buffer{}
	errorOutput := &bytes.Buffer{}
	storageTestCommand := newTestStorageCommand(
		storage,
		storageTestCommandWriters{standardOutput: output, errorOutput: errorOutput},
		"/storage/temp",
	)

	exitCode := storageTestCommand.SaveAndDeleteProbe(t.Context())

	assert.Equal(t, 0, exitCode)
	assert.True(t, strings.HasPrefix(storage.savedFileName, ".databasus-storage-test-"))
	assert.Equal(t, storage.savedFileName, storage.deletedFileName)
	assert.Equal(t, storageProbeContent, storage.savedContent)
	assert.Equal(t, "storage test passed\n", output.String())
	assert.Empty(t, errorOutput.String())
}

func Test_StorageTestCommand_WhenSaveFails_ReportsPermissionsAndRemovesTemporaryProbe(t *testing.T) {
	temporaryFolder := t.TempDir()
	storage := &fakeStorageFileSaverAndDeleter{
		saveError:       errors.New("permission denied"),
		temporaryFolder: temporaryFolder,
	}
	output := &bytes.Buffer{}
	errorOutput := &bytes.Buffer{}
	storageTestCommand := newTestStorageCommand(
		storage,
		storageTestCommandWriters{standardOutput: output, errorOutput: errorOutput},
		temporaryFolder,
	)

	exitCode := storageTestCommand.SaveAndDeleteProbe(t.Context())

	assert.Equal(t, 1, exitCode)
	assert.Empty(t, storage.deletedFileName)
	assert.Empty(t, output.String())
	assert.Contains(t, errorOutput.String(), "Required operation: save a file through local storage.")
	assert.Contains(t, errorOutput.String(), temporaryFolder+" and /storage/backups")
	assert.Contains(t, errorOutput.String(), storagePermissionsDocumentationURL)
	assert.Contains(t, errorOutput.String(), "permission denied")
	_, probeStatError := os.Stat(filepath.Join(temporaryFolder, storage.savedFileName))
	assert.ErrorIs(t, probeStatError, os.ErrNotExist)
}

func Test_StorageTestCommand_WhenDeleteFails_ReportsDeleteFailure(t *testing.T) {
	storage := &fakeStorageFileSaverAndDeleter{deleteError: errors.New("read-only filesystem")}
	errorOutput := &bytes.Buffer{}
	storageTestCommand := newTestStorageCommand(
		storage,
		storageTestCommandWriters{standardOutput: io.Discard, errorOutput: errorOutput},
		"/storage/temp",
	)

	exitCode := storageTestCommand.SaveAndDeleteProbe(t.Context())

	assert.Equal(t, 1, exitCode)
	require.NotEmpty(t, storage.savedFileName)
	assert.Equal(t, storage.savedFileName, storage.deletedFileName)
	assert.Contains(t, errorOutput.String(), "Required operation: delete a file through local storage.")
	assert.Contains(t, errorOutput.String(), "read-only filesystem")
}

func newTestStorageCommand(
	storage storageFileSaverAndDeleter,
	writers storageTestCommandWriters,
	temporaryFolder string,
) storageTestCommand {
	return storageTestCommand{
		storage:         storage,
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		writers:         writers,
		temporaryFolder: temporaryFolder,
		backupFolder:    "/storage/backups",
	}
}
