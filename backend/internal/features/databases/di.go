package databases

import (
	audit_logs "databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/notifiers"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var databaseRepository = &DatabaseRepository{}

var databaseService = &DatabaseService{
	databaseRepository,
	notifiers.GetNotifierService(),
	logger.GetLogger(),
	[]DatabaseCreationListener{},
	[]DatabaseRemoveListener{},
	[]DatabaseCopyListener{},
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
	encryption.GetFieldEncryptor(),
}

var databaseController = &DatabaseController{
	databaseService,
	users_services.GetUserService(),
	workspaces_services.GetWorkspaceService(),
}

func GetDatabaseService() *DatabaseService {
	return databaseService
}

func GetDatabaseController() *DatabaseController {
	return databaseController
}

func SetupDependencies() {
	workspaces_services.GetWorkspaceService().AddWorkspaceDeletionListener(databaseService)
	notifiers.GetNotifierService().SetNotifierDatabaseCounter(databaseService)
}
