package storages

import (
	audit_logs "databasus-backend/internal/features/audit_logs"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
)

var storageRepository = &StorageRepository{}
var storageService = &StorageService{
	storageRepository,
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
	encryption.GetFieldEncryptor(),
}
var storageController = &StorageController{
	storageService,
	workspaces_services.GetWorkspaceService(),
}

func GetStorageService() *StorageService {
	return storageService
}

func GetStorageController() *StorageController {
	return storageController
}

func SetupDependencies() {
	workspaces_services.GetWorkspaceService().AddWorkspaceDeletionListener(storageService)
}
