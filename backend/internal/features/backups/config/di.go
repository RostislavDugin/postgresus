package backups_config

import (
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/storages"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
)

var backupConfigRepository = &BackupConfigRepository{}
var backupConfigService = &BackupConfigService{
	backupConfigRepository,
	databases.GetDatabaseService(),
	storages.GetStorageService(),
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
