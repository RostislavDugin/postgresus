package restores

import (
	audit_logs "postgresus-backend/internal/features/audit_logs"
	"postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/restores/usecases"
	"postgresus-backend/internal/features/storages"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/logger"
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
