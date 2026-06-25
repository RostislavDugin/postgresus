package backuping_physical

import (
	"sync"
	"sync/atomic"

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

var physicalBackuper = &PhysicalBackuper{
	databases.GetDatabaseService(),
	encryption.GetFieldEncryptor(),
	workspaces_services.GetWorkspaceService(),
	physical_repositories.GetFullBackupRepository(),
	physical_repositories.GetIncrementalBackupRepository(),
	physical_repositories.GetInFlightBackupRepository(),
	physical_repositories.GetWalHistoryRepository(),
	backups_config_physical.GetBackupConfigService(),
	storages.GetStorageService(),
	notifiers.GetNotifierService(),
	tasks_cancellation.GetTaskCancelManager(),
	encryption_secrets.GetSecretKeyService(),
	logger.GetLogger(),
	postgresql_executor.NewCreateFullBackupUsecase(),
	postgresql_executor.NewCreateIncrementalBackupUsecase(),
}

func GetPhysicalBackuper() *PhysicalBackuper { return physicalBackuper }

var physicalBackupsScheduler = &PhysicalBackupsScheduler{
	physical_repositories.GetFullBackupRepository(),
	physical_repositories.GetIncrementalBackupRepository(),
	physical_repositories.GetInFlightBackupRepository(),
	backups_config_physical.GetBackupConfigService(),
	chain_view.GetChainViewService(),
	tasks_cancellation.GetTaskCancelManager(),
	physicalBackuper,
	atomicTime{},
	logger.GetLogger(),
	atomic.Bool{},
	atomic.Bool{},
}

func GetPhysicalBackupsScheduler() *PhysicalBackupsScheduler { return physicalBackupsScheduler }

var physicalBackupCleaner = &PhysicalBackupCleaner{
	physical_service.GetPhysicalBackupService(),
	chain_view.GetChainViewService(),
	backups_config_physical.GetBackupConfigService(),
	physical_repositories.GetFullBackupRepository(),
	physical_repositories.GetWalSegmentRepository(),
	logger.GetLogger(),
	atomic.Bool{},
}

func GetPhysicalBackupCleaner() *PhysicalBackupCleaner { return physicalBackupCleaner }

var physicalWalStreamSupervisor = &PhysicalWalStreamSupervisor{
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

func GetPhysicalWalStreamSupervisor() *PhysicalWalStreamSupervisor {
	return physicalWalStreamSupervisor
}

var physicalSlotCleanupListener = postgresql_executor.NewPhysicalSlotCleanupListener(
	databases.GetDatabaseService(),
	encryption.GetFieldEncryptor(),
	logger.GetLogger(),
)

var physicalBackupCanceller = NewPhysicalBackupCanceller(
	physical_repositories.GetInFlightBackupRepository(),
	tasks_cancellation.GetTaskCancelManager(),
	logger.GetLogger(),
)

func GetPhysicalBackupCanceller() *PhysicalBackupCanceller { return physicalBackupCanceller }

var physicalBackupCancellationListener = &PhysicalBackupCancellationListener{
	physicalBackupCanceller,
	physical_repositories.GetWalStreamerRepository(),
	tasks_cancellation.GetTaskCancelManager(),
	logger.GetLogger(),
}

var SetupDependencies = sync.OnceFunc(func() {
	// Order matters: the cancellation listener stops the local pg_receivewal and
	// deletes the streamer row first, so the slot cleanup listener that runs next
	// can drop the (now detaching) WAL slot instead of refusing it as active and
	// leaving it to pin WAL forever.
	databases.GetDatabaseService().AddDbRemoveListener(physicalBackupCancellationListener)
	databases.GetDatabaseService().AddDbRemoveListener(physicalSlotCleanupListener)
	backups_config_physical.GetBackupConfigService().SetBackupConfigChangeListener(physicalBackupCancellationListener)
})
