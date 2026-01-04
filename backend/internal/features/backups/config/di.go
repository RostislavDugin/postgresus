package backups_config

import (
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

var backupConfigRepository = &BackupConfigRepository{}
var backupConfigService = &BackupConfigService{
	backupConfigRepository,
	databases.GetDatabaseService(),
	storages.GetStorageService(),
	notifiers.GetNotifierService(),
	workspaces_services.GetWorkspaceService(),
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

func SetupDependencies() {
	storages.GetStorageService().SetStorageDatabaseCounter(backupConfigService)
}
