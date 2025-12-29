package audit_logs

import (
	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/util/logger"
)

var auditLogRepository = &AuditLogRepository{}
var auditLogService = &AuditLogService{
	auditLogRepository,
	logger.GetLogger(),
}
var auditLogController = &AuditLogController{
	auditLogService,
}
var auditLogBackgroundService = &AuditLogBackgroundService{
	auditLogService,
	logger.GetLogger(),
}

func GetAuditLogService() *AuditLogService {
	return auditLogService
}

func GetAuditLogController() *AuditLogController {
	return auditLogController
}

func GetAuditLogBackgroundService() *AuditLogBackgroundService {
	return auditLogBackgroundService
}

func SetupDependencies() {
	users_services.GetUserService().SetAuditLogWriter(auditLogService)
	users_services.GetSettingsService().SetAuditLogWriter(auditLogService)
	users_services.GetManagementService().SetAuditLogWriter(auditLogService)
}
