package audit_logs

import (
	"sync"

	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/util/logger"
)

var (
	auditLogRepository = &AuditLogRepository{}
	auditLogService    = &AuditLogService{
		auditLogRepository,
		logger.GetLogger(),
		logger.GetAuditLogger(),
	}
)

var auditLogController = &AuditLogController{
	auditLogService,
}

var auditLogBackgroundService = &AuditLogBackgroundService{
	auditLogService: auditLogService,
	logger:          logger.GetLogger(),
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

var SetupDependencies = sync.OnceFunc(func() {
	users_services.GetUserService().SetAuditLogWriter(auditLogService)
	users_services.GetSettingsService().SetAuditLogWriter(auditLogService)
	users_services.GetManagementService().SetAuditLogWriter(auditLogService)
})
