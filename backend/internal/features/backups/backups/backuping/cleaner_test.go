package backuping

import (
	"testing"
	"time"

	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/period"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_CleanOldBackups_DeletesBackupsOlderThanRetentionTimePeriod(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodWeek,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()
	oldBackup1 := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-10 * 24 * time.Hour),
	}
	oldBackup2 := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-8 * 24 * time.Hour),
	}
	recentBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-3 * 24 * time.Hour),
	}

	err = backupRepository.Save(oldBackup1)
	assert.NoError(t, err)
	err = backupRepository.Save(oldBackup2)
	assert.NoError(t, err)
	err = backupRepository.Save(recentBackup)
	assert.NoError(t, err)

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(remainingBackups))
	assert.Equal(t, recentBackup.ID, remainingBackups[0].ID)
}

func Test_CleanOldBackups_SkipsDatabaseWithForeverRetentionPeriod(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodForever,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	oldBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    time.Now().UTC().Add(-365 * 24 * time.Hour),
	}
	err = backupRepository.Save(oldBackup)
	assert.NoError(t, err)

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(remainingBackups))
	assert.Equal(t, oldBackup.ID, remainingBackups[0].ID)
}

func Test_CleanExceededBackups_WhenUnderLimit_NoBackupsDeleted(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:            database.ID,
		IsBackupsEnabled:      true,
		RetentionPolicyType:   backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod:   period.PeriodForever,
		StorageID:             &storage.ID,
		MaxBackupsTotalSizeMB: 100,
		BackupIntervalID:      interval.ID,
		BackupInterval:        interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	for i := 0; i < 3; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 16.67,
			CreatedAt:    time.Now().UTC().Add(-time.Duration(i) * time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanExceededBackups()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(remainingBackups))
}

func Test_CleanExceededBackups_WhenOverLimit_DeletesOldestBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:            database.ID,
		IsBackupsEnabled:      true,
		RetentionPolicyType:   backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod:   period.PeriodForever,
		StorageID:             &storage.ID,
		MaxBackupsTotalSizeMB: 30,
		BackupIntervalID:      interval.ID,
		BackupInterval:        interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()
	var backupIDs []uuid.UUID
	for i := 0; i < 5; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-time.Duration(4-i) * time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
		backupIDs = append(backupIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanExceededBackups()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(remainingBackups))

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}
	assert.False(t, remainingIDs[backupIDs[0]])
	assert.False(t, remainingIDs[backupIDs[1]])
	assert.True(t, remainingIDs[backupIDs[2]])
	assert.True(t, remainingIDs[backupIDs[3]])
	assert.True(t, remainingIDs[backupIDs[4]])
}

func Test_CleanExceededBackups_SkipsInProgressBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:            database.ID,
		IsBackupsEnabled:      true,
		RetentionPolicyType:   backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod:   period.PeriodForever,
		StorageID:             &storage.ID,
		MaxBackupsTotalSizeMB: 50,
		BackupIntervalID:      interval.ID,
		BackupInterval:        interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	completedBackups := make([]*backups_core.Backup, 3)
	for i := 0; i < 3; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 30,
			CreatedAt:    now.Add(-time.Duration(3-i) * time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
		completedBackups[i] = backup
	}

	inProgressBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusInProgress,
		BackupSizeMb: 10,
		CreatedAt:    now,
	}
	err = backupRepository.Save(inProgressBackup)
	assert.NoError(t, err)

	cleaner := GetBackupCleaner()
	err = cleaner.cleanExceededBackups()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(remainingBackups), 2)

	var inProgressFound bool
	for _, backup := range remainingBackups {
		if backup.ID == inProgressBackup.ID {
			inProgressFound = true
			assert.Equal(t, backups_core.BackupStatusInProgress, backup.Status)
		}
	}
	assert.True(t, inProgressFound, "In-progress backup should not be deleted")
}

func Test_CleanExceededBackups_WithZeroLimit_SkipsDatabase(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:            database.ID,
		IsBackupsEnabled:      true,
		RetentionPolicyType:   backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod:   period.PeriodForever,
		StorageID:             &storage.ID,
		MaxBackupsTotalSizeMB: 0,
		BackupIntervalID:      interval.ID,
		BackupInterval:        interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	for i := 0; i < 10; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 100,
			CreatedAt:    time.Now().UTC().Add(-time.Duration(i) * time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanExceededBackups()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(remainingBackups))
}

func Test_GetTotalSizeByDatabase_CalculatesCorrectly(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	completedBackup1 := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10.5,
		CreatedAt:    time.Now().UTC(),
	}
	completedBackup2 := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 20.3,
		CreatedAt:    time.Now().UTC(),
	}
	failedBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusFailed,
		BackupSizeMb: 5.2,
		CreatedAt:    time.Now().UTC(),
	}
	inProgressBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusInProgress,
		BackupSizeMb: 100,
		CreatedAt:    time.Now().UTC(),
	}

	err := backupRepository.Save(completedBackup1)
	assert.NoError(t, err)
	err = backupRepository.Save(completedBackup2)
	assert.NoError(t, err)
	err = backupRepository.Save(failedBackup)
	assert.NoError(t, err)
	err = backupRepository.Save(inProgressBackup)
	assert.NoError(t, err)

	totalSize, err := backupRepository.GetTotalSizeByDatabase(database.ID)
	assert.NoError(t, err)
	assert.InDelta(t, 36.0, totalSize, 0.1)
}

func Test_CleanByCount_KeepsNewestNBackups_DeletesOlder(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeCount,
		RetentionCount:      3,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()
	var backupIDs []uuid.UUID
	for i := 0; i < 5; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt: now.Add(
				-time.Duration(4-i) * time.Hour,
			), // oldest first in loop, newest = i=4
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
		backupIDs = append(backupIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(remainingBackups))

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}
	assert.False(t, remainingIDs[backupIDs[0]], "Oldest backup should be deleted")
	assert.False(t, remainingIDs[backupIDs[1]], "2nd oldest backup should be deleted")
	assert.True(t, remainingIDs[backupIDs[2]], "3rd backup should remain")
	assert.True(t, remainingIDs[backupIDs[3]], "4th backup should remain")
	assert.True(t, remainingIDs[backupIDs[4]], "Newest backup should remain")
}

func Test_CleanByCount_WhenUnderLimit_NoBackupsDeleted(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeCount,
		RetentionCount:      10,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	for i := 0; i < 5; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    time.Now().UTC().Add(-time.Duration(i) * time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(remainingBackups))
}

func Test_CleanByCount_DoesNotDeleteInProgressBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeCount,
		RetentionCount:      2,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-time.Duration(3-i) * time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
	}

	inProgressBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusInProgress,
		BackupSizeMb: 5,
		CreatedAt:    now,
	}
	err = backupRepository.Save(inProgressBackup)
	assert.NoError(t, err)

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	var inProgressFound bool
	for _, backup := range remainingBackups {
		if backup.ID == inProgressBackup.ID {
			inProgressFound = true
		}
	}
	assert.True(t, inProgressFound, "In-progress backup should not be deleted by count policy")
}

func Test_CleanByGFS_KeepsCorrectBackupsPerSlot(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeGFS,
		RetentionGfsDays:    3,
		RetentionGfsWeeks:   0,
		RetentionGfsMonths:  0,
		RetentionGfsYears:   0,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create 5 backups on 5 different days; only the 3 newest days should be kept
	var backupIDs []uuid.UUID
	for i := 0; i < 5; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-time.Duration(4-i) * 24 * time.Hour).Truncate(24 * time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
		backupIDs = append(backupIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(remainingBackups))

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}
	assert.False(t, remainingIDs[backupIDs[0]], "Oldest daily backup should be deleted")
	assert.False(t, remainingIDs[backupIDs[1]], "2nd oldest daily backup should be deleted")
	assert.True(t, remainingIDs[backupIDs[2]], "3rd backup should remain")
	assert.True(t, remainingIDs[backupIDs[3]], "4th backup should remain")
	assert.True(t, remainingIDs[backupIDs[4]], "Newest backup should remain")
}

func Test_CleanByGFS_WithWeeklyAndMonthlySlots_KeepsWiderSpread(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeGFS,
		RetentionGfsDays:    2,
		RetentionGfsWeeks:   2,
		RetentionGfsMonths:  1,
		RetentionGfsYears:   0,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create one backup per week for 6 weeks (each on Monday of that week)
	// GFS should keep: 2 daily (most recent 2 unique days) + 2 weekly + 1 monthly = up to 5 unique
	var createdIDs []uuid.UUID
	for i := 0; i < 6; i++ {
		weekOffset := time.Duration(5-i) * 7 * 24 * time.Hour
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-weekOffset).Truncate(24 * time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
		createdIDs = append(createdIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	// We should have at most 5 backups kept (2 daily + 2 weekly + 1 monthly, but with overlap possible)
	// The exact count depends on how many unique periods are covered
	assert.LessOrEqual(t, len(remainingBackups), 5)
	assert.GreaterOrEqual(t, len(remainingBackups), 1)

	// The two most recent backups should always be retained (daily slots)
	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}
	assert.True(t, remainingIDs[createdIDs[4]], "Second newest backup should be retained (daily)")
	assert.True(t, remainingIDs[createdIDs[5]], "Newest backup should be retained (daily)")
}

// Test_DeleteBackup_WhenStorageDeleteFails_BackupStillRemovedFromDatabase verifies resilience
// when storage becomes unavailable. Even if storage.DeleteFile fails (e.g., storage is offline,
// credentials changed, or storage was deleted), the backup record should still be removed from
// the database. This prevents orphaned backup records when storage is no longer accessible.
func Test_DeleteBackup_WhenStorageDeleteFails_BackupStillRemovedFromDatabase(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	backup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    testStorage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    time.Now().UTC(),
	}
	err := backupRepository.Save(backup)
	assert.NoError(t, err)

	cleaner := GetBackupCleaner()

	err = cleaner.DeleteBackup(backup)
	assert.NoError(t, err, "DeleteBackup should succeed even when storage file doesn't exist")

	deletedBackup, err := backupRepository.FindByID(backup.ID)
	assert.Error(t, err, "Backup should not exist in database")
	assert.Nil(t, deletedBackup)
}

func Test_CleanByGFS_WithHourlySlots_KeepsCorrectBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeGFS,
		RetentionGfsHours:   3,
		StorageID:           &testStorage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create 5 backups spaced 1 hour apart; only the 3 newest hours should be kept
	var backupIDs []uuid.UUID
	for i := 0; i < 5; i++ {
		backup := &backups_core.Backup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    testStorage.ID,
			Status:       backups_core.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-time.Duration(4-i) * time.Hour).Truncate(time.Hour),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)
		backupIDs = append(backupIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(remainingBackups))

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}
	assert.False(t, remainingIDs[backupIDs[0]], "Oldest hourly backup should be deleted")
	assert.False(t, remainingIDs[backupIDs[1]], "2nd oldest hourly backup should be deleted")
	assert.True(t, remainingIDs[backupIDs[2]], "3rd backup should remain")
	assert.True(t, remainingIDs[backupIDs[3]], "4th backup should remain")
	assert.True(t, remainingIDs[backupIDs[4]], "Newest backup should remain")
}

func Test_BuildGFSKeepSet(t *testing.T) {
	// Fixed reference time: a Wednesday mid-month to avoid boundary edge cases in the default tests.
	// Use time.Date for determinism across test runs.
	ref := time.Date(2025, 6, 18, 12, 0, 0, 0, time.UTC) // Wednesday, 2025-06-18

	day := 24 * time.Hour
	week := 7 * day

	newBackup := func(createdAt time.Time) *backups_core.Backup {
		return &backups_core.Backup{ID: uuid.New(), CreatedAt: createdAt}
	}

	// backupsEveryDay returns n backups, newest-first, each 1 day apart.
	backupsEveryDay := func(n int) []*backups_core.Backup {
		bs := make([]*backups_core.Backup, n)
		for i := 0; i < n; i++ {
			bs[i] = newBackup(ref.Add(-time.Duration(i) * day))
		}
		return bs
	}

	// backupsEveryWeek returns n backups, newest-first, each 7 days apart.
	backupsEveryWeek := func(n int) []*backups_core.Backup {
		bs := make([]*backups_core.Backup, n)
		for i := 0; i < n; i++ {
			bs[i] = newBackup(ref.Add(-time.Duration(i) * week))
		}
		return bs
	}

	hour := time.Hour

	// backupsEveryHour returns n backups, newest-first, each 1 hour apart.
	backupsEveryHour := func(n int) []*backups_core.Backup {
		bs := make([]*backups_core.Backup, n)
		for i := 0; i < n; i++ {
			bs[i] = newBackup(ref.Add(-time.Duration(i) * hour))
		}
		return bs
	}

	tests := []struct {
		name         string
		backups      []*backups_core.Backup
		hours        int
		days         int
		weeks        int
		months       int
		years        int
		keptIndices  []int   // which indices in backups should be kept
		deletedRange *[2]int // optional: all indices in [from, to) must be deleted
	}{
		{
			name:        "OnlyHourlySlots_KeepsNewest3Of5",
			backups:     backupsEveryHour(5),
			hours:       3,
			keptIndices: []int{0, 1, 2},
		},
		{
			name: "SameHourDedup_OnlyNewestKeptForHourlySlot",
			backups: []*backups_core.Backup{
				newBackup(ref.Truncate(hour).Add(45 * time.Minute)),
				newBackup(ref.Truncate(hour).Add(10 * time.Minute)),
			},
			hours:       1,
			keptIndices: []int{0},
		},
		{
			name:        "OnlyDailySlots_KeepsNewest3Of5",
			backups:     backupsEveryDay(5),
			days:        3,
			keptIndices: []int{0, 1, 2},
		},
		{
			name:        "OnlyDailySlots_FewerBackupsThanSlots_KeepsAll",
			backups:     backupsEveryDay(2),
			days:        5,
			keptIndices: []int{0, 1},
		},
		{
			name:        "OnlyWeeklySlots_KeepsNewest2Weeks",
			backups:     backupsEveryWeek(4),
			weeks:       2,
			keptIndices: []int{0, 1},
		},
		{
			name: "OnlyMonthlySlots_KeepsNewest2Months",
			backups: []*backups_core.Backup{
				newBackup(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)),
			},
			months:      2,
			keptIndices: []int{0, 1},
		},
		{
			name: "OnlyYearlySlots_KeepsNewest2Years",
			backups: []*backups_core.Backup{
				newBackup(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)),
			},
			years:       2,
			keptIndices: []int{0, 1},
		},
		{
			name: "SameDayDedup_OnlyNewestKeptForDailySlot",
			backups: []*backups_core.Backup{
				// Two backups on the same day; newest-first order
				newBackup(ref.Truncate(day).Add(10 * time.Hour)),
				newBackup(ref.Truncate(day).Add(2 * time.Hour)),
			},
			days:        1,
			keptIndices: []int{0},
		},
		{
			name: "SameWeekDedup_OnlyNewestKeptForWeeklySlot",
			backups: []*backups_core.Backup{
				// ref is Wednesday; add Thursday of same week
				newBackup(ref.Add(1 * day)), // Thursday same week
				newBackup(ref),              // Wednesday same week
			},
			weeks:       1,
			keptIndices: []int{0},
		},
		{
			name: "AdditiveSlots_NewestFillsDailyAndWeeklyAndMonthly",
			// Newest backup fills daily + weekly + monthly simultaneously
			backups: []*backups_core.Backup{
				newBackup(time.Date(2025, 6, 18, 12, 0, 0, 0, time.UTC)), // newest
				newBackup(time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)), // 1 week ago
				newBackup(time.Date(2025, 5, 18, 12, 0, 0, 0, time.UTC)), // 1 month ago
				newBackup(time.Date(2025, 4, 18, 12, 0, 0, 0, time.UTC)), // 2 months ago
			},
			days:        1,
			weeks:       2,
			months:      2,
			keptIndices: []int{0, 1, 2},
		},
		{
			name: "YearBoundary_CorrectlySplitsAcrossYears",
			backups: []*backups_core.Backup{
				newBackup(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)),
			},
			years:       2,
			keptIndices: []int{0, 1}, // 2025 and 2024 kept; 2024-06 and 2023 deleted
		},
		{
			name: "ISOWeekBoundary_Jan1UsesCorrectISOWeek",
			// 2025-01-01 is ISO week 1 of 2025; 2024-12-28 is ISO week 52 of 2024
			backups: []*backups_core.Backup{
				newBackup(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),   // ISO week 2025-W01
				newBackup(time.Date(2024, 12, 28, 12, 0, 0, 0, time.UTC)), // ISO week 2024-W52
			},
			weeks:       2,
			keptIndices: []int{0, 1}, // different ISO weeks → both kept
		},
		{
			name:        "EmptyBackups_ReturnsEmptyKeepSet",
			backups:     []*backups_core.Backup{},
			hours:       3,
			days:        3,
			weeks:       2,
			months:      1,
			years:       1,
			keptIndices: []int{},
		},
		{
			name:        "AllZeroSlots_KeepsNothing",
			backups:     backupsEveryDay(5),
			hours:       0,
			days:        0,
			weeks:       0,
			months:      0,
			years:       0,
			keptIndices: []int{},
		},
		{
			name:    "AllSlotsActive_FullCombination",
			backups: backupsEveryWeek(12),
			days:    2,
			weeks:   3,
			months:  2,
			years:   1,
			// 2 daily (indices 0,1) + 3rd weekly slot (index 2) + 2nd monthly slot (index 3 or later).
			// Additive slots: newest fills daily+weekly+monthly+yearly; each subsequent week fills another weekly,
			// and a backup ~4 weeks later fills the 2nd monthly slot.
			keptIndices: []int{0, 1, 2, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keepSet := buildGFSKeepSet(tc.backups, tc.hours, tc.days, tc.weeks, tc.months, tc.years)

			keptIndexSet := make(map[int]bool, len(tc.keptIndices))
			for _, idx := range tc.keptIndices {
				keptIndexSet[idx] = true
			}

			for i, backup := range tc.backups {
				if keptIndexSet[i] {
					assert.True(t, keepSet[backup.ID], "backup at index %d should be kept", i)
				} else {
					assert.False(t, keepSet[backup.ID], "backup at index %d should be deleted", i)
				}
			}
		})
	}
}

func Test_CleanByTimePeriod_SkipsRecentBackup_EvenIfOlderThanRetention(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	// Retention period is 1 day — any backup older than 1 day should be deleted.
	// But the recent backup was created only 30 minutes ago and must be preserved.
	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodDay,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	oldBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-2 * 24 * time.Hour),
	}
	recentBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-30 * time.Minute),
	}

	err = backupRepository.Save(oldBackup)
	assert.NoError(t, err)
	err = backupRepository.Save(recentBackup)
	assert.NoError(t, err)

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(remainingBackups))
	assert.Equal(t, recentBackup.ID, remainingBackups[0].ID)
}

func Test_CleanByCount_SkipsRecentBackup_EvenIfOverLimit(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	// Retention count is 2 — 4 backups exist so 2 should be deleted.
	// The oldest backup in the "excess" tail was made 30 min ago — it must be preserved.
	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeCount,
		RetentionCount:      2,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	oldBackup1 := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-5 * time.Hour),
	}
	oldBackup2 := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-3 * time.Hour),
	}
	// This backup is 3rd newest and would normally be deleted — but it is recent.
	recentExcessBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-30 * time.Minute),
	}
	newestBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-10 * time.Minute),
	}

	for _, b := range []*backups_core.Backup{oldBackup1, oldBackup2, recentExcessBackup, newestBackup} {
		err = backupRepository.Save(b)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}

	assert.False(t, remainingIDs[oldBackup1.ID], "Oldest non-recent backup should be deleted")
	assert.False(t, remainingIDs[oldBackup2.ID], "2nd oldest non-recent backup should be deleted")
	assert.True(
		t,
		remainingIDs[recentExcessBackup.ID],
		"Recent backup must be preserved despite being over limit",
	)
	assert.True(t, remainingIDs[newestBackup.ID], "Newest backup should be preserved")
}

func Test_CleanByGFS_SkipsRecentBackup_WhenNotInKeepSet(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	// Keep only 1 daily slot. We create 2 old backups plus two recent backups on today.
	// Backups are ordered newest-first, so the 15-min-old backup fills the single daily slot.
	// The 30-min-old backup is the same day → not in the GFS keep-set, but it is still recent
	// (within grace period) and must be preserved.
	backupConfig := &backups_config.BackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config.RetentionPolicyTypeGFS,
		RetentionGfsDays:    1,
		StorageID:           &storage.ID,
		BackupIntervalID:    interval.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	oldBackup1 := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-3 * 24 * time.Hour).Truncate(24 * time.Hour),
	}
	oldBackup2 := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-2 * 24 * time.Hour).Truncate(24 * time.Hour),
	}
	// Newest backup today — will fill the single GFS daily slot.
	newestTodayBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-15 * time.Minute),
	}
	// Slightly older backup, also today — NOT in GFS keep-set (duplicate day),
	// but within the 60-minute grace period so it must survive.
	recentNotInKeepSet := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-30 * time.Minute),
	}

	for _, b := range []*backups_core.Backup{oldBackup1, oldBackup2, newestTodayBackup, recentNotInKeepSet} {
		err = backupRepository.Save(b)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}

	assert.False(t, remainingIDs[oldBackup1.ID], "Old backup 1 should be deleted by GFS")
	assert.False(t, remainingIDs[oldBackup2.ID], "Old backup 2 should be deleted by GFS")
	assert.True(
		t,
		remainingIDs[newestTodayBackup.ID],
		"Newest backup fills GFS daily slot and must remain",
	)
	assert.True(
		t,
		remainingIDs[recentNotInKeepSet.ID],
		"Recent backup not in keep-set must be preserved by grace period",
	)
}

func Test_CleanExceededBackups_SkipsRecentBackup_WhenOverTotalSizeLimit(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
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
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	interval := createTestInterval()

	// Total size limit is 10 MB. We have two backups of 8 MB each (16 MB total).
	// The oldest backup was created 30 minutes ago — within the grace period.
	// The cleaner must stop and leave both backups intact.
	backupConfig := &backups_config.BackupConfig{
		DatabaseID:            database.ID,
		IsBackupsEnabled:      true,
		RetentionPolicyType:   backups_config.RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod:   period.PeriodForever,
		StorageID:             &storage.ID,
		MaxBackupsTotalSizeMB: 10,
		BackupIntervalID:      interval.ID,
		BackupInterval:        interval,
	}
	_, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	olderRecentBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 8,
		CreatedAt:    now.Add(-30 * time.Minute),
	}
	newerRecentBackup := &backups_core.Backup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core.BackupStatusCompleted,
		BackupSizeMb: 8,
		CreatedAt:    now.Add(-10 * time.Minute),
	}

	err = backupRepository.Save(olderRecentBackup)
	assert.NoError(t, err)
	err = backupRepository.Save(newerRecentBackup)
	assert.NoError(t, err)

	cleaner := GetBackupCleaner()
	err = cleaner.cleanExceededBackups()
	assert.NoError(t, err)

	remainingBackups, err := backupRepository.FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(
		t,
		2,
		len(remainingBackups),
		"Both recent backups must be preserved even though total size exceeds limit",
	)
}

// Mock listener for testing
type mockBackupRemoveListener struct {
	onBeforeBackupRemove func(*backups_core.Backup) error
}

func (m *mockBackupRemoveListener) OnBeforeBackupRemove(backup *backups_core.Backup) error {
	if m.onBeforeBackupRemove != nil {
		return m.onBeforeBackupRemove(backup)
	}

	return nil
}

func createTestInterval() *intervals.Interval {
	timeOfDay := "04:00"
	interval := &intervals.Interval{
		Interval:  intervals.IntervalDaily,
		TimeOfDay: &timeOfDay,
	}

	err := storage.GetDb().Create(interval).Error
	if err != nil {
		panic(err)
	}

	return interval
}
