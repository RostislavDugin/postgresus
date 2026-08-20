package backuping_logical

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/features/storages"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	cache_utils "databasus-backend/internal/util/cache"
	util_logger "databasus-backend/internal/util/logger"
)

func Test_BackupExecuted_NotificationSent(t *testing.T) {
	cache_utils.ClearAllCache()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, storage)

	defer func() {
		// cleanup backups first
		backups, _ := backupRepository.FindByDatabaseID(database.ID)
		for _, backup := range backups {
			backupRepository.DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond) // Wait for cascading deletes
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
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
			mock.MatchedBy(func(notification notifier_models.Notification) bool {
				return notification.Type == notifier_models.NotificationTypeBackupFailed &&
					strings.Contains(notification.Heading, "❌ Backup failed") &&
					strings.Contains(notification.Message, "backup failed")
			}),
		).Once()

		backuper.MakeBackup(t.Context(), backup.ID, true)

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
			mock.MatchedBy(func(notification notifier_models.Notification) bool {
				return notification.Type == notifier_models.NotificationTypeBackupSuccess &&
					strings.Contains(notification.Heading, "✅ Backup completed") &&
					strings.Contains(notification.Message, "Backup completed successfully")
			}),
		).Once()

		backuper.MakeBackup(t.Context(), backup.ID, true)

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
		var capturedNotification notifier_models.Notification

		mockNotificationSender.On("SendNotification",
			mock.Anything,
			mock.AnythingOfType("notifier_models.Notification"),
		).Run(func(args mock.Arguments) {
			capturedNotifier = args.Get(0).(*notifiers.Notifier)
			capturedNotification = args.Get(1).(notifier_models.Notification)
		}).Once()

		backuper.MakeBackup(t.Context(), backup.ID, true)

		// Verify expectations were met
		mockNotificationSender.AssertExpectations(t)

		// Additional detailed assertions
		assert.Equal(t, notifier_models.NotificationTypeBackupSuccess, capturedNotification.Type)
		assert.Contains(t, capturedNotification.Heading, "✅ Backup completed")
		assert.Contains(t, capturedNotification.Heading, database.Name)
		assert.Contains(t, capturedNotification.Message, "Backup completed successfully")
		assert.Contains(t, capturedNotification.Message, "10.00 MB")
		assert.Equal(t, notifier.ID, capturedNotifier.ID)
	})
}

func Test_MakeBackup_WhenCallerContextCarriesRequestID_FinishLineIsAttributed(t *testing.T) {
	fixture := CreateBackupTestFixture(t, "Attribution Test Workspace")

	capturingHandler := &requestIDCapturingHandler{requestIDByMessage: map[string]string{}}

	mockNotificationSender := &MockNotificationSender{}
	mockNotificationSender.On("SendNotification", mock.Anything, mock.Anything).Maybe()

	backuper := CreateTestBackuper()
	backuper.notificationSender = mockNotificationSender
	backuper.createBackupUseCase = &CreateSuccessBackupUsecase{}
	backuper.logger = slog.New(capturingHandler)

	backup := SeedInProgressTestBackup(t, fixture.Database.ID, fixture.Storage.ID)

	const requestID = "request-id-under-test"

	backuper.MakeBackup(util_logger.ContextWithRequestID(t.Context(), requestID), backup.ID, true)

	capturedRequestID, isLogged := capturingHandler.GetRequestIDForMessagePrefix("logical backup finished")

	assert.True(t, isLogged, "the backup finish line was never logged")
	assert.Equal(t, requestID, capturedRequestID)
}

// A record logged on the detached execution context is indistinguishable from one that never
// carried a request at all, which is what this pins.
type requestIDCapturingHandler struct {
	mutex              sync.Mutex
	requestIDByMessage map[string]string
}

func (h *requestIDCapturingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *requestIDCapturingHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.requestIDByMessage[record.Message] = util_logger.GetRequestID(ctx)

	return nil
}

func (h *requestIDCapturingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *requestIDCapturingHandler) WithGroup(string) slog.Handler { return h }

func (h *requestIDCapturingHandler) GetRequestIDForMessagePrefix(prefix string) (string, bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for message, requestID := range h.requestIDByMessage {
		if strings.HasPrefix(message, prefix) {
			return requestID, true
		}
	}

	return "", false
}
