package backups_config

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	plans "databasus-backend/internal/features/plan"
	"databasus-backend/internal/features/storages"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var backupConfigRepository = &BackupConfigRepository{}
var backupConfigService = &BackupConfigService{
	backupConfigRepository,
	databases.GetDatabaseService(),
	storages.GetStorageService(),
	notifiers.GetNotifierService(),
	workspaces_services.GetWorkspaceService(),
	plans.GetDatabasePlanService(),
	nil,
}
var backupConfigController = &BackupConfigController{
	backupConfigService,
}

func GetBackupConfigController() *BackupConfigController {
	return backupConfigController
}

func GetBackupConfigService() *BackupConfigService {
	return backupConfigService
}

func GetBackupConfigRepository() *BackupConfigRepository {
	return backupConfigRepository
}

var (
	setupOnce sync.Once
	isSetup   atomic.Bool
)

func SetupDependencies() {
	wasAlreadySetup := isSetup.Load()

	setupOnce.Do(func() {
		storages.GetStorageService().SetStorageDatabaseCounter(backupConfigService)

		isSetup.Store(true)
	})

	if wasAlreadySetup {
		logger.GetLogger().Warn("SetupDependencies called multiple times, ignoring subsequent call")
	}
}
