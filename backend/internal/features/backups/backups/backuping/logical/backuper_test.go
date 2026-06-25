package backuping_logical

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	cache_utils "databasus-backend/internal/util/cache"
)

func Test_BackupExecuted_NotificationSent(t *testing.T) {
	cache_utils.ClearAllCache()
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(database.ID, storage)

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
		backuper := CreateTestBackuper()
		backuper.notificationSender = mockNotificationSender
		backuper.createBackupUseCase = &CreateFailedBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core_logical.LogicalBackup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core_logical.BackupStatusInProgress,
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

		backuper.MakeBackup(backup.ID, true)

		// Verify all expectations were met
		mockNotificationSender.AssertExpectations(t)
	})

	t.Run("BackupSuccess_SuccessNotificationSent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backuper := CreateTestBackuper()
		backuper.notificationSender = mockNotificationSender
		backuper.createBackupUseCase = &CreateSuccessBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core_logical.LogicalBackup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core_logical.BackupStatusInProgress,
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

		backuper.MakeBackup(backup.ID, true)

		// Verify all expectations were met
		mockNotificationSender.AssertExpectations(t)
	})

	t.Run("BackupSuccess_VerifyNotificationContent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backuper := CreateTestBackuper()
		backuper.notificationSender = mockNotificationSender
		backuper.createBackupUseCase = &CreateSuccessBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core_logical.LogicalBackup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			Status:     backups_core_logical.BackupStatusInProgress,
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

		backuper.MakeBackup(backup.ID, true)

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
