package notifiers

import (
	audit_logs "databasus-backend/internal/features/audit_logs"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var notifierRepository = &NotifierRepository{}
var notifierService = &NotifierService{
	notifierRepository,
	logger.GetLogger(),
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
	encryption.GetFieldEncryptor(),
	nil,
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
