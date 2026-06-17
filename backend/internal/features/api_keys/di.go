package api_keys

import (
	audit_logs "databasus-backend/internal/features/audit_logs"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/logger"
)

var apiKeyRepository = &ApiKeyRepository{}

var apiKeyService = &ApiKeyService{
	apiKeyRepository,
	backups_services.GetBackupService(),
	databases.GetDatabaseService(),
	audit_logs.GetAuditLogService(),
	logger.GetLogger(),
}

func GetApiKeyService() *ApiKeyService { return apiKeyService }
