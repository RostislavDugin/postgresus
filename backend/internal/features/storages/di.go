package storages

import (
	audit_logs "postgresus-backend/internal/features/audit_logs"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
)

var storageRepository = &StorageRepository{}
var storageService = &StorageService{
	storageRepository,
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
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
