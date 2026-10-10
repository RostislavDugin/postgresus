package backups_core_logical

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	storage_files "databasus-backend/internal/features/storages/files"
	test_utils "databasus-backend/internal/util/testing"
)

// FailingBackupFileStore fails every write with FailureErr. It either drains the whole stream
// first, so the dump runs to completion before the storage fails, or fails before reading a
// byte, so the storage fails while the dump is still producing data.
type FailingBackupFileStore struct {
	FailureErr             error
	IsFailingBeforeReading bool

	readBytes atomic.Int64
}

func (s *FailingBackupFileStore) WriteFile(
	_ context.Context,
	_ storage_files.StoredFileReference,
	file io.Reader,
) (storage_files.WriteReceipt, error) {
	if !s.IsFailingBeforeReading {
		readBytes, _ := io.Copy(io.Discard, file)
		s.readBytes.Store(readBytes)
	}

	return storage_files.WriteReceipt{}, s.FailureErr
}

func (s *FailingBackupFileStore) GetReadBytes() int64 {
	return s.readBytes.Load()
}

// ExecuteBackupInto runs one engine's real backup use case against a test database, writing into
// the given file store.
type ExecuteBackupInto func(
	ctx context.Context,
	backup *LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	fileStore BackupFileStore,
) error

// AssertStorageTimeoutReportedAsStorageFailure runs an engine's real dump into a storage that
// times out, once after the dump finished and once while it is still running, and checks the
// engine reports the storage error rather than a cancellation.
func AssertStorageTimeoutReportedAsStorageFailure(t *testing.T, executeBackupInto ExecuteBackupInto) {
	t.Helper()

	storageErr := fmt.Errorf("failed to upload file to S3: %w", test_utils.GetNetworkTimeoutError(t))

	t.Run("Test_Execute_WhenStorageTimesOutAfterDumpFinished_ReturnsStorageError", func(t *testing.T) {
		fileStore := &FailingBackupFileStore{FailureErr: storageErr}

		backupErr := executeBackupInto(
			t.Context(),
			newStorageFailureTestBackup(),
			newUnencryptedTestBackupConfig(),
			fileStore,
		)

		require.Positive(t, fileStore.GetReadBytes(), "the dump never reached the storage")
		assertStorageErrorReported(t, backupErr, storageErr)
	})

	t.Run("Test_Execute_WhenStorageFailsBeforeReading_ReturnsStorageError", func(t *testing.T) {
		fileStore := &FailingBackupFileStore{FailureErr: storageErr, IsFailingBeforeReading: true}

		backupErr := executeBackupInto(
			t.Context(),
			newStorageFailureTestBackup(),
			newUnencryptedTestBackupConfig(),
			fileStore,
		)

		assertStorageErrorReported(t, backupErr, storageErr)
	})
}

func assertStorageErrorReported(t *testing.T, backupErr, storageErr error) {
	t.Helper()

	require.Error(t, backupErr)
	assert.True(t, errors.Is(backupErr, storageErr), "the backup error must wrap the storage error, got: %v", backupErr)
	assert.Contains(t, backupErr.Error(), "save to storage")
	assert.NotContains(t, backupErr.Error(), "backup cancelled")
}

func newStorageFailureTestBackup() *LogicalBackup {
	backupID := uuid.New()

	return &LogicalBackup{
		ID:         backupID,
		DatabaseID: uuid.New(),
		StorageID:  uuid.New(),
		FileName:   "storage-failure-" + backupID.String(),
		Status:     BackupStatusInProgress,
	}
}

func newUnencryptedTestBackupConfig() *backups_config_logical.LogicalBackupConfig {
	return &backups_config_logical.LogicalBackupConfig{Encryption: backups_core_enums.BackupEncryptionNone}
}
