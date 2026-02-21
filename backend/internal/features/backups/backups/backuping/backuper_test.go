package backuping

import (
	"strings"
	"testing"
	"time"

	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	cache_utils "databasus-backend/internal/util/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_BackupExecuted_NotificationSent(t *testing.T) {
	cache_utils.ClearAllCache()
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	backups_config.EnableBackupsForTestDatabase(database.ID, storage)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(database)
		time.Sleep(50 * time.Millisecond) // Wait for cascading deletes
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	t.Run("BackupFailed_FailNotificationSent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backuperNode := CreateTestBackuperNode()
		backuperNode.notificationSender = mockNotificationSender
		backuperNode.createBackupUseCase = &CreateFailedBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core.Backup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err := backupRepository.Save(backup)
		assert.NoError(t, err)

		// Set up expectations
		mockNotificationSender.On("SendNotification",
			mock.Anything,
			mock.MatchedBy(func(title string) bool {
				return strings.Contains(title, "❌ Backup failed")
			}),
			mock.MatchedBy(func(message string) bool {
				return strings.Contains(message, "backup failed")
			}),
		).Once()

		backuperNode.MakeBackup(backup.ID, true)

		// Verify all expectations were met
		mockNotificationSender.AssertExpectations(t)
	})

	t.Run("BackupSuccess_SuccessNotificationSent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backuperNode := CreateTestBackuperNode()
		backuperNode.notificationSender = mockNotificationSender
		backuperNode.createBackupUseCase = &CreateSuccessBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core.Backup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err := backupRepository.Save(backup)
		assert.NoError(t, err)

		// Set up expectations
		mockNotificationSender.On("SendNotification",
			mock.Anything,
			mock.MatchedBy(func(title string) bool {
				return strings.Contains(title, "✅ Backup completed")
			}),
			mock.MatchedBy(func(message string) bool {
				return strings.Contains(message, "Backup completed successfully")
			}),
		).Once()

		backuperNode.MakeBackup(backup.ID, true)

		// Verify all expectations were met
		mockNotificationSender.AssertExpectations(t)
	})

	t.Run("BackupSuccess_VerifyNotificationContent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backuperNode := CreateTestBackuperNode()
		backuperNode.notificationSender = mockNotificationSender
		backuperNode.createBackupUseCase = &CreateSuccessBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core.Backup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err := backupRepository.Save(backup)
		assert.NoError(t, err)

		// capture arguments
		var capturedNotifier *notifiers.Notifier
		var capturedTitle string
		var capturedMessage string

		mockNotificationSender.On("SendNotification",
			mock.Anything,
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
		).Run(func(args mock.Arguments) {
			capturedNotifier = args.Get(0).(*notifiers.Notifier)
			capturedTitle = args.Get(1).(string)
			capturedMessage = args.Get(2).(string)
		}).Once()

		backuperNode.MakeBackup(backup.ID, true)

		// Verify expectations were met
		mockNotificationSender.AssertExpectations(t)

		// Additional detailed assertions
		assert.Contains(t, capturedTitle, "✅ Backup completed")
		assert.Contains(t, capturedTitle, database.Name)
		assert.Contains(t, capturedMessage, "Backup completed successfully")
		assert.Contains(t, capturedMessage, "10.00 MB")
		assert.Equal(t, notifier.ID, capturedNotifier.ID)
	})
}

func Test_BackupSizeLimits(t *testing.T) {
	cache_utils.ClearAllCache()
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
		time.Sleep(50 * time.Millisecond) // Wait for cascading deletes
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(storage.ID)
		workspaces_testing.RemoveTestWorkspace(workspace, router)
	}()

	t.Run("UnlimitedSize_MaxBackupSizeMBIsZero_BackupCompletes", func(t *testing.T) {
		// Enable backups with unlimited size (0)
		backupConfig := backups_config.EnableBackupsForTestDatabase(database.ID, storage)
		backupConfig.MaxBackupSizeMB = 0 // unlimited
		backupConfig, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
		assert.NoError(t, err)

		backuperNode := CreateTestBackuperNode()
		backuperNode.createBackupUseCase = &CreateLargeBackupUsecase{}

		// Create a backup record
		backup := &backups_core.Backup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)

		backuperNode.MakeBackup(backup.ID, false)

		// Verify backup completed successfully even with large size
		updatedBackup, err := backupRepository.FindByID(backup.ID)
		assert.NoError(t, err)
		assert.Equal(t, backups_core.BackupStatusCompleted, updatedBackup.Status)
		assert.Equal(t, float64(10000), updatedBackup.BackupSizeMb)
		assert.Nil(t, updatedBackup.FailMessage)
	})

	t.Run("SizeExceeded_BackupFailedWithIsSkipRetry", func(t *testing.T) {
		// Enable backups with 5 MB limit
		backupConfig := backups_config.EnableBackupsForTestDatabase(database.ID, storage)
		backupConfig.MaxBackupSizeMB = 5
		backupConfig, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
		assert.NoError(t, err)

		backuperNode := CreateTestBackuperNode()
		backuperNode.createBackupUseCase = &CreateProgressiveBackupUsecase{}

		// Create a backup record
		backup := &backups_core.Backup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)

		backuperNode.MakeBackup(backup.ID, false)

		// Verify backup was marked as failed with IsSkipRetry=true
		updatedBackup, err := backupRepository.FindByID(backup.ID)
		assert.NoError(t, err)
		assert.Equal(t, backups_core.BackupStatusFailed, updatedBackup.Status)
		assert.True(t, updatedBackup.IsSkipRetry)
		assert.NotNil(t, updatedBackup.FailMessage)
		assert.Contains(t, *updatedBackup.FailMessage, "exceeded maximum allowed size")
		assert.Contains(t, *updatedBackup.FailMessage, "10.00 MB")
		assert.Contains(t, *updatedBackup.FailMessage, "5 MB")
		assert.Greater(t, updatedBackup.BackupSizeMb, float64(5))
	})

	t.Run("SizeWithinLimit_BackupCompletes", func(t *testing.T) {
		// Enable backups with 100 MB limit
		backupConfig := backups_config.EnableBackupsForTestDatabase(database.ID, storage)
		backupConfig.MaxBackupSizeMB = 100
		backupConfig, err := backups_config.GetBackupConfigRepository().Save(backupConfig)
		assert.NoError(t, err)

		backuperNode := CreateTestBackuperNode()
		backuperNode.createBackupUseCase = &CreateMediumBackupUsecase{}

		// Create a backup record
		backup := &backups_core.Backup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err = backupRepository.Save(backup)
		assert.NoError(t, err)

		backuperNode.MakeBackup(backup.ID, false)

		// Verify backup completed successfully
		updatedBackup, err := backupRepository.FindByID(backup.ID)
		assert.NoError(t, err)
		assert.Equal(t, backups_core.BackupStatusCompleted, updatedBackup.Status)
		assert.Equal(t, float64(50), updatedBackup.BackupSizeMb)
		assert.Nil(t, updatedBackup.FailMessage)
	})
}
