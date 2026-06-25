package backups_services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
)

func insertInProgressBackup(t *testing.T) *backups_core_logical.LogicalBackup {
	t.Helper()

	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace, err := workspaces_testing.CreateTestWorkspaceDirect("wait-test-workspace", owner.UserID)
	require.NoError(t, err)

	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	backup := &backups_core_logical.LogicalBackup{
		ID:         uuid.New(),
		DatabaseID: database.ID,
		StorageID:  storage.ID,
		Status:     backups_core_logical.BackupStatusInProgress,
		CreatedAt:  time.Now().UTC(),
	}
	backup.GenerateFilename("waittest")

	repo := backups_core_logical.GetBackupRepository()
	require.NoError(t, repo.Save(backup))

	t.Cleanup(func() {
		_ = repo.DeleteByID(backup.ID)
		databases.RemoveTestDatabase(database)
		// RemoveTestDatabase triggers an async cascade (backup config cleanup);
		// wait briefly so dependent resources (storage, notifier) are not removed first.
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		_ = workspaces_testing.RemoveTestWorkspaceDirect(workspace.ID)
	})

	return backup
}

func Test_WaitForBackupTerminal_WhenBackupCompletes_ReturnsCompleted(t *testing.T) {
	backup := insertInProgressBackup(t)
	repo := backups_core_logical.GetBackupRepository()

	go func() {
		time.Sleep(500 * time.Millisecond)
		backup.Status = backups_core_logical.BackupStatusCompleted
		_ = repo.Save(backup)
	}()

	finished, err := GetBackupService().waitForBackupTerminal(t.Context(), backup.ID, 30*time.Second)

	require.NoError(t, err)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, finished.Status)
}

func Test_WaitForBackupTerminal_WhenBackupStaysInProgress_ReturnsTimeout(t *testing.T) {
	backup := insertInProgressBackup(t)

	finished, err := GetBackupService().waitForBackupTerminal(t.Context(), backup.ID, 1*time.Second)

	assert.ErrorIs(t, err, ErrBackupWaitTimeout)
	assert.Equal(t, backups_core_logical.BackupStatusInProgress, finished.Status)
}
