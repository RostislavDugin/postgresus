package backups

import (
	"errors"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/features/storages"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_testing "postgresus-backend/internal/features/users/testing"
	workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
	"postgresus-backend/internal/util/logger"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_BackupExecuted_NotificationSent(t *testing.T) {
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
		backupService := &BackupService{
			databases.GetDatabaseService(),
			storages.GetStorageService(),
			backupRepository,
			notifiers.GetNotifierService(),
			mockNotificationSender,
			backups_config.GetBackupConfigService(),
			&CreateFailedBackupUsecase{},
			logger.GetLogger(),
			[]BackupRemoveListener{},
			nil, // workspaceService
			nil, // auditLogService
		}

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

		backupService.MakeBackup(database.ID, true)

		// Verify all expectations were met
		mockNotificationSender.AssertExpectations(t)
	})

	t.Run("BackupSuccess_SuccessNotificationSent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}

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

		backupService := &BackupService{
			databases.GetDatabaseService(),
			storages.GetStorageService(),
			backupRepository,
			notifiers.GetNotifierService(),
			mockNotificationSender,
			backups_config.GetBackupConfigService(),
			&CreateSuccessBackupUsecase{},
			logger.GetLogger(),
			[]BackupRemoveListener{},
			nil, // workspaceService
			nil, // auditLogService
		}

		backupService.MakeBackup(database.ID, true)

		// Verify all expectations were met
		mockNotificationSender.AssertExpectations(t)
	})

	t.Run("BackupSuccess_VerifyNotificationContent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backupService := &BackupService{
			databases.GetDatabaseService(),
			storages.GetStorageService(),
			backupRepository,
			notifiers.GetNotifierService(),
			mockNotificationSender,
			backups_config.GetBackupConfigService(),
			&CreateSuccessBackupUsecase{},
			logger.GetLogger(),
			[]BackupRemoveListener{},
			nil, // workspaceService
			nil, // auditLogService
		}

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

		backupService.MakeBackup(database.ID, true)

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

type CreateFailedBackupUsecase struct {
}

func (uc *CreateFailedBackupUsecase) Execute(
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(
		completedMBs float64,
	),
) error {
	backupProgressListener(10) // Assume we completed 10MB
	return errors.New("backup failed")
}

type CreateSuccessBackupUsecase struct {
}

func (uc *CreateSuccessBackupUsecase) Execute(
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(
		completedMBs float64,
	),
) error {
	backupProgressListener(10) // Assume we completed 10MB
	return nil
}
