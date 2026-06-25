package backuping_logical

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	usecases_logical "databasus-backend/internal/features/backups/backups/usecases/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func seedBackup(
	t *testing.T,
	label string,
	backup *backups_core_logical.LogicalBackup,
) *backups_core_logical.LogicalBackup {
	t.Helper()

	if err := storage.GetDb().Create(backup).Error; err != nil {
		t.Fatalf("seed %s backup: %v", label, err)
	}

	return backup
}

func SeedTestBackup(
	t *testing.T,
	databaseID, storageID uuid.UUID,
	sizeMb float64,
) *backups_core_logical.LogicalBackup {
	t.Helper()

	return seedBackup(t, "completed", &backups_core_logical.LogicalBackup{
		ID:                uuid.New(),
		FileName:          "test-backup-" + uuid.New().String(),
		DatabaseID:        databaseID,
		StorageID:         storageID,
		Status:            backups_core_logical.BackupStatusCompleted,
		BackupSizeMb:      sizeMb,
		BackupRawDbSizeMb: sizeMb,
		CreatedAt:         time.Now().UTC(),
	})
}

// SeedInProgressTestBackup inserts an IN_PROGRESS backup row so MakeBackup can
// pick it up and drive it through to completion (mirrors backuper_test.go's
// manual setup, but reusable across packages).
func SeedInProgressTestBackup(
	t *testing.T,
	databaseID, storageID uuid.UUID,
) *backups_core_logical.LogicalBackup {
	t.Helper()

	return seedBackup(t, "in-progress", &backups_core_logical.LogicalBackup{
		ID:         uuid.New(),
		DatabaseID: databaseID,
		StorageID:  storageID,
		Status:     backups_core_logical.BackupStatusInProgress,
		CreatedAt:  time.Now().UTC(),
	})
}

func CreateTestRouter() *gin.Engine {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config_logical.GetBackupConfigController(),
	)

	return router
}

func CreateTestBackupCleaner() *BackupCleaner {
	return &BackupCleaner{
		backupRepository,
		storages.GetStorageService(),
		backups_config_logical.GetBackupConfigService(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		[]backups_core_logical.BackupRemoveListener{},
		atomic.Bool{},
	}
}

func CreateTestBackuper() *Backuper {
	return &Backuper{
		databases.GetDatabaseService(),
		encryption.GetFieldEncryptor(),
		workspaces_services.GetWorkspaceService(),
		backupRepository,
		backups_config_logical.GetBackupConfigService(),
		storages.GetStorageService(),
		notifiers.GetNotifierService(),
		taskCancelManager,
		logger.GetLogger(),
		usecases_logical.GetCreateBackupUsecase(),
	}
}

func CreateTestBackuperWithUseCase(useCase backups_core_logical.CreateBackupUsecase) *Backuper {
	return &Backuper{
		databases.GetDatabaseService(),
		encryption.GetFieldEncryptor(),
		workspaces_services.GetWorkspaceService(),
		backupRepository,
		backups_config_logical.GetBackupConfigService(),
		storages.GetStorageService(),
		notifiers.GetNotifierService(),
		taskCancelManager,
		logger.GetLogger(),
		useCase,
	}
}

func CreateTestScheduler() *BackupsScheduler {
	return CreateTestSchedulerWithBackuper(CreateTestBackuper())
}

// CreateTestSchedulerWithBackuper wires the scheduler to a specific backuper, so
// a test can inject a backuper with a mock use case and have StartBackup dispatch
// to it in-process.
func CreateTestSchedulerWithBackuper(backuper *Backuper) *BackupsScheduler {
	return &BackupsScheduler{
		backupRepository,
		backups_config_logical.GetBackupConfigService(),
		taskCancelManager,
		databases.GetDatabaseService(),
		time.Now().UTC(),
		logger.GetLogger(),
		backuper,
		[]backups_core_logical.BackupCompletionListener{},
		atomic.Bool{},
		atomic.Bool{},
	}
}

// WaitForBackupCompletion waits for a new backup to be created and completed (or failed)
// for the given database. It checks for backups with count greater than expectedInitialCount.
func WaitForBackupCompletion(
	t *testing.T,
	databaseID uuid.UUID,
	expectedInitialCount int,
	timeout time.Duration,
) {
	deadline := time.Now().UTC().Add(timeout)

	for time.Now().UTC().Before(deadline) {
		backups, err := backupRepository.FindByDatabaseID(databaseID)
		if err != nil {
			t.Logf("WaitForBackupCompletion: error finding backups: %v", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}

		t.Logf(
			"WaitForBackupCompletion: found %d backups (expected > %d)",
			len(backups),
			expectedInitialCount,
		)

		if len(backups) > expectedInitialCount {
			// Check if the newest backup has completed or failed
			newestBackup := backups[0]
			t.Logf("WaitForBackupCompletion: newest backup status: %s", newestBackup.Status)

			if newestBackup.Status == backups_core_logical.BackupStatusCompleted ||
				newestBackup.Status == backups_core_logical.BackupStatusFailed ||
				newestBackup.Status == backups_core_logical.BackupStatusCanceled {
				t.Logf(
					"WaitForBackupCompletion: backup finished with status %s",
					newestBackup.Status,
				)
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("WaitForBackupCompletion: timeout waiting for backup to complete")
}

// StartSchedulerForTest starts the BackupsScheduler in a goroutine for testing.
// The scheduler subscribes to task completions and manages backup lifecycle.
// Returns a context cancel function that should be deferred to stop the scheduler.
//
// PubSubManager.Subscribe handshakes with Valkey before returning, so we
// don't need to sleep here waiting for the subscription to register. Poll
// the scheduler's hasRun flag instead to be sure Run() has entered its
// loop before the caller proceeds.
func StartSchedulerForTest(t *testing.T, scheduler *BackupsScheduler) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		scheduler.Run(ctx)
		close(done)
	}()

	deadline := time.Now().UTC().Add(5 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if scheduler.IsRunning() {
			t.Log("BackupsScheduler started")

			return func() {
				cancel()
				select {
				case <-done:
					t.Log("BackupsScheduler stopped gracefully")
				case <-time.After(2 * time.Second):
					t.Log("BackupsScheduler stop timeout")
				}
			}
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatal("BackupsScheduler failed to start within timeout")

	return nil
}

// TriggerBackupCompletedForTest fires the DI scheduler's backup-completion
// listeners for a backup, standing in for the in-process completion hook that
// StartBackup's dispatch goroutine calls after MakeBackup returns.
func TriggerBackupCompletedForTest(backupID uuid.UUID) {
	backupsScheduler.onBackupCompleted(backupID)
}
