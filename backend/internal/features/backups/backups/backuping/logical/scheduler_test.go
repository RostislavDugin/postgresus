package backuping_logical

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	cache_utils "databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/period"
	"databasus-backend/internal/util/testing/containers"
)

func Test_RunPendingBackups_ByDatabaseType_OnlySchedulesNonAgentManagedBackups(t *testing.T) {
	type testCase struct {
		name             string
		createDatabase   func(t *testing.T, workspaceID uuid.UUID, storage *storages.Storage, notifier *notifiers.Notifier) *databases.Database
		isBackupExpected bool
	}

	testCases := []testCase{
		{
			name: "PostgreSQL PG_DUMP - backup runs",
			createDatabase: func(_ *testing.T, workspaceID uuid.UUID, storage *storages.Storage, notifier *notifiers.Notifier) *databases.Database {
				return databases.CreateTestDatabase(workspaceID, storage, notifier)
			},
			isBackupExpected: true,
		},
		{
			name: "MariaDB - backup runs",
			createDatabase: func(t *testing.T, workspaceID uuid.UUID, _ *storages.Storage, notifier *notifiers.Notifier) *databases.Database {
				endpoint := containers.StartMariadb(t, "mariadb:10.11")
				return databases.CreateTestMariadbDatabase(endpoint.Host, endpoint.Port, workspaceID, notifier)
			},
			isBackupExpected: true,
		},
		{
			name: "MongoDB - backup runs",
			createDatabase: func(t *testing.T, workspaceID uuid.UUID, _ *storages.Storage, notifier *notifiers.Notifier) *databases.Database {
				endpoint := containers.StartMongodb(t, "mongo:7.0")
				return databases.CreateTestMongodbDatabase(endpoint.Host, endpoint.Port, workspaceID, notifier)
			},
			isBackupExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache_utils.ClearAllCache()

			user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
			router := CreateTestRouter()
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
			storage := storages.CreateTestStorage(workspace.ID)
			notifier := notifiers.CreateTestNotifier(workspace.ID)
			database := tc.createDatabase(t, workspace.ID, storage, notifier)

			defer func() {
				backups, _ := backupRepository.FindByDatabaseID(database.ID)
				for _, backup := range backups {
					backupRepository.DeleteByID(backup.ID)
				}

				databases.RemoveTestDatabase(database)
				time.Sleep(50 * time.Millisecond)
				storages.RemoveTestStorage(storage.ID)
				notifiers.RemoveTestNotifier(notifier)
				workspaces_testing.RemoveTestWorkspace(workspace, router)
			}()

			backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
			assert.NoError(t, err)

			timeOfDay := "04:00"
			backupConfig.BackupInterval = intervals.Interval{
				Type:      intervals.IntervalDaily,
				TimeOfDay: &timeOfDay,
			}
			backupConfig.IsBackupsEnabled = true
			backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
			backupConfig.RetentionTimePeriod = period.PeriodWeek
			backupConfig.Storage = storage
			backupConfig.StorageID = &storage.ID

			_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
			assert.NoError(t, err)

			// add old backup (24h ago)
			backupRepository.Save(&backups_core_logical.LogicalBackup{
				DatabaseID: database.ID,
				StorageID:  storage.ID,
				Status:     backups_core_logical.BackupStatusCompleted,
				CreatedAt:  time.Now().UTC().Add(-24 * time.Hour),
			})

			GetBackupsScheduler().runPendingBackups()

			if tc.isBackupExpected {
				WaitForBackupCompletion(t, database.ID, 1, 10*time.Second)

				backups, err := backupRepository.FindByDatabaseID(database.ID)
				assert.NoError(t, err)
				assert.Len(t, backups, 2)
			} else {
				time.Sleep(100 * time.Millisecond)

				backups, err := backupRepository.FindByDatabaseID(database.ID)
				assert.NoError(t, err)
				assert.Len(t, backups, 1)
			}

			time.Sleep(200 * time.Millisecond)
		})
	}
}

func Test_RunPendingBackups_WhenLastBackupWasYesterday_CreatesNewBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database
	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// add old backup
	backupRepository.Save(&backups_core_logical.LogicalBackup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status: backups_core_logical.BackupStatusCompleted,

		CreatedAt: time.Now().UTC().Add(-24 * time.Hour),
	})

	GetBackupsScheduler().runPendingBackups()

	// Wait for backup to complete (runs in goroutine)
	WaitForBackupCompletion(t, database.ID, 1, 10*time.Second)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 2)

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenLastBackupWasRecentlyCompleted_SkipsBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database
	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	backupRepository.Save(&backups_core_logical.LogicalBackup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status: backups_core_logical.BackupStatusCompleted,

		CreatedAt: time.Now().UTC(),
	})

	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1) // Should still be 1 backup, no new backup created

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenLastBackupFailedAndRetriesDisabled_SkipsBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database with retries disabled
	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID
	backupConfig.IsRetryIfFailed = false
	backupConfig.MaxFailedTriesCount = 0

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	failMessage := "backup failed"
	backupRepository.Save(&backups_core_logical.LogicalBackup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status:      backups_core_logical.BackupStatusFailed,
		FailMessage: &failMessage,

		CreatedAt: time.Now().UTC(),
	})

	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1) // Should still be 1 backup, no retry attempted

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenLastBackupFailedAndRetriesEnabled_CreatesNewBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database with retries enabled
	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID
	backupConfig.IsRetryIfFailed = true
	backupConfig.MaxFailedTriesCount = 3

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// add failed backup
	failMessage := "backup failed"
	backupRepository.Save(&backups_core_logical.LogicalBackup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status:      backups_core_logical.BackupStatusFailed,
		FailMessage: &failMessage,

		CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
	})

	GetBackupsScheduler().runPendingBackups()

	// Wait for backup to complete (runs in goroutine)
	WaitForBackupCompletion(t, database.ID, 1, 10*time.Second)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 2) // Should have 2 backups, retry was attempted

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenFailedBackupsExceedMaxRetries_SkipsBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	// setup data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for the database with retries enabled
	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID
	backupConfig.IsRetryIfFailed = true
	backupConfig.MaxFailedTriesCount = 3

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	failMessage := "backup failed"

	for range 3 {
		backupRepository.Save(&backups_core_logical.LogicalBackup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,

			Status:      backups_core_logical.BackupStatusFailed,
			FailMessage: &failMessage,

			CreatedAt: time.Now().UTC(),
		})
	}

	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	// assertions
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 3) // Should have 3 backups, not more than max

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenBackupsDisabled_SkipsBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = false
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// add old backup that would trigger new backup if enabled
	backupRepository.Save(&backups_core_logical.LogicalBackup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status: backups_core_logical.BackupStatusCompleted,

		CreatedAt: time.Now().UTC().Add(-24 * time.Hour),
	})

	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)

	// Wait for any cleanup operations to complete before defer cleanup runs
	time.Sleep(200 * time.Millisecond)
}

func Test_FailBackupsInProgress_WhenSchedulerStarts_CancelsBackupsAndUpdatesStatus(t *testing.T) {
	cache_utils.ClearAllCache()

	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)

		cache_utils.ClearAllCache()
	}()

	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// Create two in-progress backups that should be failed on scheduler restart
	backup1 := &backups_core_logical.LogicalBackup{
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core_logical.BackupStatusInProgress,
		BackupSizeMb: 10.5,
		CreatedAt:    time.Now().UTC().Add(-30 * time.Minute),
	}
	err = backupRepository.Save(backup1)
	assert.NoError(t, err)

	backup2 := &backups_core_logical.LogicalBackup{
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core_logical.BackupStatusInProgress,
		BackupSizeMb: 5.2,
		CreatedAt:    time.Now().UTC().Add(-15 * time.Minute),
	}
	err = backupRepository.Save(backup2)
	assert.NoError(t, err)

	// Create a completed backup to verify it's not affected by failBackupsInProgress
	completedBackup := &backups_core_logical.LogicalBackup{
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core_logical.BackupStatusCompleted,
		BackupSizeMb: 20.0,
		CreatedAt:    time.Now().UTC().Add(-1 * time.Hour),
	}
	err = backupRepository.Save(completedBackup)
	assert.NoError(t, err)

	// Trigger the scheduler's failBackupsInProgress logic
	// This should cancel in-progress backups and mark them as failed
	err = GetBackupsScheduler().failBackupsInProgress()
	assert.NoError(t, err)

	// Verify all backups exist and were processed correctly
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 3)

	var failedCount int
	var completedCount int
	for _, backup := range backups {
		switch backup.Status {
		case backups_core_logical.BackupStatusFailed:
			failedCount++
			// Verify fail message indicates application restart
			assert.NotNil(t, backup.FailMessage)
			assert.Equal(t, "Backup failed due to application restart", *backup.FailMessage)
			// Verify backup size was reset to 0
			assert.Equal(t, float64(0), backup.BackupSizeMb)
		case backups_core_logical.BackupStatusCompleted:
			completedCount++
		}
	}

	// Verify correct number of backups in each state
	assert.Equal(t, 2, failedCount)
	assert.Equal(t, 1, completedCount)

	time.Sleep(200 * time.Millisecond)
}

func Test_StartBackup_WhenBackupAlreadyInProgress_SkipsNewBackup(t *testing.T) {
	cache_utils.ClearAllCache()
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// Create an in-progress backup manually
	inProgressBackup := &backups_core_logical.LogicalBackup{
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core_logical.BackupStatusInProgress,
		BackupSizeMb: 0,
		CreatedAt:    time.Now().UTC(),
	}
	err = backupRepository.Save(inProgressBackup)
	assert.NoError(t, err)

	// Try to start a new backup - should be skipped
	GetBackupsScheduler().StartBackup(database, false)

	time.Sleep(200 * time.Millisecond)

	// Verify only 1 backup exists (the original in-progress one)
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1)
	assert.Equal(t, backups_core_logical.BackupStatusInProgress, backups[0].Status)
	assert.Equal(t, inProgressBackup.ID, backups[0].ID)

	time.Sleep(200 * time.Millisecond)
}

func Test_RunPendingBackups_WhenLastBackupFailedWithIsSkipRetry_SkipsBackupEvenWithRetriesEnabled(
	t *testing.T,
) {
	cache_utils.ClearAllCache()
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups with retries enabled and high retry count
	backupConfig, err := backups_config_logical.GetBackupConfigService().GetBackupConfigByDbId(database.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig.IsBackupsEnabled = true
	backupConfig.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig.RetentionTimePeriod = period.PeriodWeek
	backupConfig.Storage = storage
	backupConfig.StorageID = &storage.ID
	backupConfig.IsRetryIfFailed = true
	backupConfig.MaxFailedTriesCount = 5

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig)
	assert.NoError(t, err)

	// Create a failed backup with IsSkipRetry set to true
	failMessage := "backup failed due to size limit exceeded"
	backupRepository.Save(&backups_core_logical.LogicalBackup{
		DatabaseID: database.ID,
		StorageID:  storage.ID,

		Status:      backups_core_logical.BackupStatusFailed,
		FailMessage: &failMessage,
		IsSkipRetry: true,

		CreatedAt: time.Now().UTC(),
	})

	// Verify GetRemainedBackupTryCount returns 0 even though retries are enabled
	lastBackup, err := backupRepository.FindLastByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.NotNil(t, lastBackup)

	remainedTries := GetBackupsScheduler().GetRemainedBackupTryCount(lastBackup)
	assert.Equal(t, 0, remainedTries, "Should return 0 tries when IsSkipRetry is true")

	// Run the scheduler
	GetBackupsScheduler().runPendingBackups()

	time.Sleep(100 * time.Millisecond)

	// Verify no new backup was created (still only 1 backup exists)
	backups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Len(t, backups, 1, "No retry should be attempted when IsSkipRetry is true")

	time.Sleep(200 * time.Millisecond)
}

func Test_StartBackup_When2BackupsStartedForDifferentDatabases_BothUseCasesAreCalled(t *testing.T) {
	cache_utils.ClearAllCache()

	// Create mock tracking use case
	mockUseCase := NewMockTrackingBackupUsecase()

	// Create Backuper with mock use case and wire it into the scheduler so
	// StartBackup dispatches to the mock in-process.
	backuper := CreateTestBackuperWithUseCase(mockUseCase)

	// Create scheduler
	scheduler := CreateTestSchedulerWithBackuper(backuper)
	schedulerCancel := StartSchedulerForTest(t, scheduler)
	defer schedulerCancel()

	// Setup test data
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	// Create 2 separate databases
	database1 := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	database2 := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		// Cleanup backups for database1
		backups1, _ := backupRepository.FindByDatabaseID(database1.ID)
		for _, backup := range backups1 {
			backupRepository.DeleteByID(backup.ID)
		}

		// Cleanup backups for database2
		backups2, _ := backupRepository.FindByDatabaseID(database2.ID)
		for _, backup := range backups2 {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database1)
		databases.RemoveTestDatabase(database2)
		time.Sleep(50 * time.Millisecond)
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	// Enable backups for database1
	backupConfig1, err := backups_config_logical.GetBackupConfigService().
		GetBackupConfigByDbId(database1.ID)
	assert.NoError(t, err)

	timeOfDay := "04:00"
	backupConfig1.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig1.IsBackupsEnabled = true
	backupConfig1.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig1.RetentionTimePeriod = period.PeriodWeek
	backupConfig1.Storage = storage
	backupConfig1.StorageID = &storage.ID

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig1)
	assert.NoError(t, err)

	// Enable backups for database2
	backupConfig2, err := backups_config_logical.GetBackupConfigService().
		GetBackupConfigByDbId(database2.ID)
	assert.NoError(t, err)

	backupConfig2.BackupInterval = intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}
	backupConfig2.IsBackupsEnabled = true
	backupConfig2.RetentionPolicyType = backups_config_logical.RetentionPolicyTypeTimePeriod
	backupConfig2.RetentionTimePeriod = period.PeriodWeek
	backupConfig2.Storage = storage
	backupConfig2.StorageID = &storage.ID

	_, err = backups_config_logical.GetBackupConfigService().SaveBackupConfig(backupConfig2)
	assert.NoError(t, err)

	// Start 2 backups simultaneously
	t.Log("Starting backup for database1")
	scheduler.StartBackup(database1, false)

	t.Log("Starting backup for database2")
	scheduler.StartBackup(database2, false)

	// Wait up to 10 seconds for both backups to complete
	t.Log("Waiting for both backups to complete...")

	success := assert.Eventually(t, func() bool {
		callCount := mockUseCase.GetCallCount()
		t.Logf("Current call count: %d/2", callCount)
		return callCount == 2
	}, 10*time.Second, 200*time.Millisecond, "Both use cases should be called within 10 seconds")

	if !success {
		t.Logf("Test failed: Only %d out of 2 use cases were called", mockUseCase.GetCallCount())
	}

	// Verify both backup IDs were received
	calledBackupIDs := mockUseCase.GetCalledBackupIDs()
	t.Logf("Called backup IDs: %v", calledBackupIDs)
	assert.Len(t, calledBackupIDs, 2, "Both backup IDs should be tracked")

	// The mock records the call at use-case entry, before MakeBackup persists the
	// terminal status, so wait for both dispatch goroutines to finish flipping the
	// backups to COMPLETED before asserting on their status.
	assert.Eventually(t, func() bool {
		dbBackups1, _ := backupRepository.FindByDatabaseID(database1.ID)
		dbBackups2, _ := backupRepository.FindByDatabaseID(database2.ID)

		return len(dbBackups1) == 1 &&
			dbBackups1[0].Status == backups_core_logical.BackupStatusCompleted &&
			len(dbBackups2) == 1 &&
			dbBackups2[0].Status == backups_core_logical.BackupStatusCompleted
	}, 10*time.Second, 100*time.Millisecond, "both backups should reach COMPLETED")

	// Verify both backups exist in repository and are completed
	backups1, err := backupRepository.FindByDatabaseID(database1.ID)
	assert.NoError(t, err)
	assert.Len(t, backups1, 1, "Database1 should have 1 backup")
	if len(backups1) > 0 {
		t.Logf("Database1 backup status: %s", backups1[0].Status)
	}

	backups2, err := backupRepository.FindByDatabaseID(database2.ID)
	assert.NoError(t, err)
	assert.Len(t, backups2, 1, "Database2 should have 1 backup")
	if len(backups2) > 0 {
		t.Logf("Database2 backup status: %s", backups2[0].Status)
	}

	// Verify both backups completed successfully
	if len(backups1) > 0 {
		assert.Equal(t, backups_core_logical.BackupStatusCompleted, backups1[0].Status,
			"Database1 backup should be completed")
	}

	if len(backups2) > 0 {
		assert.Equal(t, backups_core_logical.BackupStatusCompleted, backups2[0].Status,
			"Database2 backup should be completed")
	}

	time.Sleep(200 * time.Millisecond)
}
