package backups_config_physical

import (
	"sync"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

var (
	backupConfigRepository = &BackupConfigRepository{}
	backupConfigService    = &BackupConfigService{
		backupConfigRepository,
		databases.GetDatabaseService(),
		storages.GetStorageService(),
		notifiers.GetNotifierService(),
		workspaces_services.GetWorkspaceService(),
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
})
