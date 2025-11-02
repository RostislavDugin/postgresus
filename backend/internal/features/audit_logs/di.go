package audit_logs

import (
	users_services "postgresus-backend/internal/features/users/services"
	"postgresus-backend/internal/util/logger"
)

var auditLogRepository = &AuditLogRepository{}
var auditLogService = &AuditLogService{
	auditLogRepository: auditLogRepository,
	logger:             logger.GetLogger(),
}
var auditLogController = &AuditLogController{
	auditLogService: auditLogService,
}

func GetAuditLogService() *AuditLogService {
	return auditLogService
}

func GetAuditLogController() *AuditLogController {
	return auditLogController
}

func SetupDependencies() {
	users_services.GetUserService().SetAuditLogWriter(auditLogService)
	users_services.GetSettingsService().SetAuditLogWriter(auditLogService)
	users_services.GetManagementService().SetAuditLogWriter(auditLogService)
}
