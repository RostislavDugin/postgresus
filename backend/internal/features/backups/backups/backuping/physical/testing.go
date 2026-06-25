package backuping_physical

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

// CreateTestPhysicalBackuper returns a fully wired PhysicalBackuper for
// tests. The notification sender is parameterized so tests that don't want
// to exercise the notifier stack can inject a counting / no-op stub.
// Pass nil to use the production notifier service.
func CreateTestPhysicalBackuper(notificationSender NotificationSender) *PhysicalBackuper {
	sender := notificationSender
	if sender == nil {
		sender = notifiers.GetNotifierService()
	}

	return &PhysicalBackuper{
		databases.GetDatabaseService(),
		encryption.GetFieldEncryptor(),
		workspaces_services.GetWorkspaceService(),
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetInFlightBackupRepository(),
		physical_repositories.GetWalHistoryRepository(),
		backups_config_physical.GetBackupConfigService(),
		storages.GetStorageService(),
		sender,
		tasks_cancellation.GetTaskCancelManager(),
		encryption_secrets.GetSecretKeyService(),
		logger.GetLogger(),
		postgresql_executor.NewCreateFullBackupUsecase(),
		postgresql_executor.NewCreateIncrementalBackupUsecase(),
	}
}

// CreateTestPhysicalScheduler returns a scheduler wired to the production repos
// and its own in-process backuper.
func CreateTestPhysicalScheduler() *PhysicalBackupsScheduler {
	return &PhysicalBackupsScheduler{
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetInFlightBackupRepository(),
		backups_config_physical.GetBackupConfigService(),
		chain_view.GetChainViewService(),
		tasks_cancellation.GetTaskCancelManager(),
		CreateTestPhysicalBackuper(nil),
		atomicTime{},
		logger.GetLogger(),
		atomic.Bool{},
		atomic.Bool{},
	}
}

// CreateTestWalStreamSupervisor returns a fresh WAL stream supervisor wired to
// the production repos and services. A fresh instance (not a copy of the DI
// singleton) keeps each test's hasRun/running state isolated and avoids copying
// the embedded mutex.
func CreateTestWalStreamSupervisor() *PhysicalWalStreamSupervisor {
	return &PhysicalWalStreamSupervisor{
		databases.GetDatabaseService(),
		backups_config_physical.GetBackupConfigService(),
		storages.GetStorageService(),
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalHistoryRepository(),
		physical_repositories.GetWalStreamerRepository(),
		notifiers.GetNotifierService(),
		tasks_cancellation.GetTaskCancelManager(),
		encryption_secrets.GetSecretKeyService(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		sync.Mutex{},
		make(map[uuid.UUID]*runningStreamer),
		atomicTime{},
		atomic.Bool{},
		atomic.Bool{},
	}
}

// CreateTestPhysicalCleaner returns a cleaner wired to the production service +
// repos.
func CreateTestPhysicalCleaner() *PhysicalBackupCleaner {
	return &PhysicalBackupCleaner{
		physical_service.GetPhysicalBackupService(),
		chain_view.GetChainViewService(),
		backups_config_physical.GetBackupConfigService(),
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetWalSegmentRepository(),
		logger.GetLogger(),
		atomic.Bool{},
	}
}

// StartPhysicalSchedulerForTest starts a fresh scheduler's Run loop in a
// goroutine and waits until it is ready (IsRunning). The scheduler invokes its
// own in-process backuper directly, so a backup requested over HTTP (config-enable
// bootstrap for the FULL, the trigger endpoint for incrementals) is picked up on
// the next tick and run to a terminal state. Returns a cancel func the caller
// defers; it cancels and waits for the goroutine to drain.
func StartPhysicalSchedulerForTest(t *testing.T) context.CancelFunc {
	t.Helper()

	scheduler := CreateTestPhysicalScheduler()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		scheduler.Run(ctx)
		close(done)
	}()

	deadline := time.Now().UTC().Add(5 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if scheduler.IsRunning() {
			return func() {
				cancel()

				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Log("physical scheduler stop timeout")
				}
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("physical scheduler failed to start within timeout")

	return nil
}

// StartPhysicalWalStreamSupervisorForTest starts a fresh WAL stream supervisor's
// Run loop in a goroutine and waits until it is ready (IsRunning). It mirrors the
// supervisor wiring in cmd/main.go: with the supervisor up, enabling
// WAL-stream backups over the HTTP config API causes it to claim the database and
// create its replication slot on the next reconcile tick — so tests drive the slot
// lifecycle through the API instead of starting a streamer by hand. Returns a
// cancel func the caller defers; it cancels and waits for the goroutine to drain.
func StartPhysicalWalStreamSupervisorForTest(t *testing.T) context.CancelFunc {
	t.Helper()

	supervisor := CreateTestWalStreamSupervisor()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		supervisor.Run(ctx)
		close(done)
	}()

	deadline := time.Now().UTC().Add(5 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if supervisor.IsRunning() {
			return func() {
				cancel()

				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Log("physical wal stream supervisor stop timeout")
				}
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("physical wal stream supervisor failed to start within timeout")

	return nil
}
