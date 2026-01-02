package restores

import (
	audit_logs "databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/backups/backups"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/disk"
	"databasus-backend/internal/features/restores/usecases"
	"databasus-backend/internal/features/storages"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var restoreRepository = &RestoreRepository{}
var restoreService = &RestoreService{
	backups.GetBackupService(),
	restoreRepository,
	storages.GetStorageService(),
	backups_config.GetBackupConfigService(),
	usecases.GetRestoreBackupUsecase(),
	databases.GetDatabaseService(),
	logger.GetLogger(),
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
	encryption.GetFieldEncryptor(),
	disk.GetDiskService(),
}
var restoreController = &RestoreController{
	restoreService,
}

var restoreBackgroundService = &RestoreBackgroundService{
	restoreRepository,
	logger.GetLogger(),
}

func GetRestoreController() *RestoreController {
	return restoreController
}

func GetRestoreBackgroundService() *RestoreBackgroundService {
	return restoreBackgroundService
}

func SetupDependencies() {
	backups.GetBackupService().AddBackupRemoveListener(restoreService)
}
