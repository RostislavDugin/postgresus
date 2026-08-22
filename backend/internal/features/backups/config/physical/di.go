package backups_config_physical

import (
	"sync"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var (
	backupConfigRepository = &BackupConfigRepository{}
	backupConfigService    = &BackupConfigService{
		backupConfigRepository,
		databases.GetDatabaseService(),
		storages.GetStorageService(),
		notifiers.GetNotifierService(),
		workspaces_services.GetWorkspaceService(),
		logger.GetLogger(),
		nil,
		nil,
	}
)

var backupConfigController = &BackupConfigController{
	backupConfigService,
}

func GetBackupConfigController() *BackupConfigController {
	return backupConfigController
}

func GetBackupConfigService() *BackupConfigService {
	return backupConfigService
}

var SetupDependencies = sync.OnceFunc(func() {
	storages.GetStorageService().AddStorageDatabaseCounter(backupConfigService)
	databases.GetDatabaseService().AddDbBackupTypeChangeListener(backupConfigService)
})
