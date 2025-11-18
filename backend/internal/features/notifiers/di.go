package notifiers

import (
	audit_logs "postgresus-backend/internal/features/audit_logs"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/logger"
)

var notifierRepository = &NotifierRepository{}
var notifierService = &NotifierService{
	notifierRepository,
	logger.GetLogger(),
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
	encryption.GetFieldEncryptor(),
}
var notifierController = &NotifierController{
	notifierService,
	workspaces_services.GetWorkspaceService(),
}

func GetNotifierController() *NotifierController {
	return notifierController
}

func GetNotifierService() *NotifierService {
	return notifierService
}

func GetNotifierRepository() *NotifierRepository {
	return notifierRepository
}
func SetupDependencies() {
	workspaces_services.GetWorkspaceService().AddWorkspaceDeletionListener(notifierService)
}
