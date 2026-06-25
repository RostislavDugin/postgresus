package restoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/config"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	"databasus-backend/internal/features/notifiers"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	cache_utils "databasus-backend/internal/util/cache"
)

func Test_MakeRestore_WhenCacheMissed_RestoreFails(t *testing.T) {
	cache_utils.ClearAllCache()

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(database.ID, storage)

	defer func() {
		backupRepo := backups_core_logical.BackupRepository{}
		backupsList, _ := backupRepo.FindByDatabaseID(database.ID)
		for _, backup := range backupsList {
			backupRepo.DeleteByID(backup.ID)
		}

		restoreRepo := restores_core.RestoreRepository{}
		restoresInProgress, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusInProgress)
		for _, restore := range restoresInProgress {
			restoreRepo.DeleteByID(restore.ID)
		}
		restoresFailed, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusFailed)
		for _, restore := range restoresFailed {
			restoreRepo.DeleteByID(restore.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)

		cache_utils.ClearAllCache()
	}()

	backup := backups_controllers_logical.CreateTestBackup(database.ID, storage.ID)

	// Create restore but DON'T cache DB credentials
	// Also don't set embedded DB fields to avoid schema issues
	restore := &restores_core.Restore{
		BackupID: backup.ID,
		Status:   restores_core.RestoreStatusInProgress,
	}
	err := restoreRepository.Save(restore)
	assert.NoError(t, err)

	// Create restorer and execute restore (should fail due to cache miss)
	restorer := CreateTestRestorer()
	restorer.MakeRestore(restore.ID)

	// Verify restore failed with appropriate error message
	updatedRestore, err := restoreRepository.FindByID(restore.ID)
	assert.NoError(t, err)
	assert.Equal(t, restores_core.RestoreStatusFailed, updatedRestore.Status)
	assert.NotNil(t, updatedRestore.FailMessage)
	assert.Contains(
		t,
		*updatedRestore.FailMessage,
		"Database credentials expired or missing from cache",
	)
}

func Test_MakeRestore_WhenTaskStarts_CacheDeletedImmediately(t *testing.T) {
	cache_utils.ClearAllCache()

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(database.ID, storage)

	defer func() {
		backupRepo := backups_core_logical.BackupRepository{}
		backupsList, _ := backupRepo.FindByDatabaseID(database.ID)
		for _, backup := range backupsList {
			backupRepo.DeleteByID(backup.ID)
		}

		restoreRepo := restores_core.RestoreRepository{}
		restoresInProgress, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusInProgress)
		for _, restore := range restoresInProgress {
			restoreRepo.DeleteByID(restore.ID)
		}
		restoresFailed, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusFailed)
		for _, restore := range restoresFailed {
			restoreRepo.DeleteByID(restore.ID)
		}
		restoresCompleted, _ := restoreRepo.FindByStatus(restores_core.RestoreStatusCompleted)
		for _, restore := range restoresCompleted {
			restoreRepo.DeleteByID(restore.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)

		cache_utils.ClearAllCache()
	}()

	backup := backups_controllers_logical.CreateTestBackup(database.ID, storage.ID)

	// Create restore with cached DB credentials
	// Don't set embedded DB fields in the restore model itself
	restore := &restores_core.Restore{
		BackupID: backup.ID,
		Status:   restores_core.RestoreStatusInProgress,
	}
	err := restoreRepository.Save(restore)
	assert.NoError(t, err)

	// Cache DB credentials separately
	dbCache := &RestoreDatabaseCache{
		PostgresqlLogicalDatabase: &postgresql_logical.PostgresqlLogicalDatabase{
			Host:     config.GetEnv().TestLocalhost,
			Port:     5432,
			Username: "test",
			Password: "test",
			Database: new("testdb"),
			Version:  "16",
		},
	}
	restoreDatabaseCache.SetWithExpiration(restore.ID.String(), dbCache, 1*time.Hour)

	// Verify cache exists before restore starts
	cachedDB := restoreDatabaseCache.Get(restore.ID.String())
	assert.NotNil(t, cachedDB, "Cache should exist before restore starts")

	// Start restore (this will call GetAndDelete)
	restorer := CreateTestRestorer()
	restorer.MakeRestore(restore.ID)

	// Verify cache was deleted immediately
	cachedDBAfter := restoreDatabaseCache.Get(restore.ID.String())
	assert.Nil(t, cachedDBAfter, "Cache should be deleted immediately when task starts")
}
