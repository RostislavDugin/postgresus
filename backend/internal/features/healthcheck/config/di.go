package healthcheck_config

import (
	"postgresus-backend/internal/features/audit_logs"
	"postgresus-backend/internal/features/databases"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"postgresus-backend/internal/util/logger"
)

var healthcheckConfigRepository = &HealthcheckConfigRepository{}
var healthcheckConfigService = &HealthcheckConfigService{
	databases.GetDatabaseService(),
	healthcheckConfigRepository,
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
	logger.GetLogger(),
}
var healthcheckConfigController = &HealthcheckConfigController{
	healthcheckConfigService,
}

func GetHealthcheckConfigService() *HealthcheckConfigService {
	return healthcheckConfigService
}

func GetHealthcheckConfigController() *HealthcheckConfigController {
	return healthcheckConfigController
}

func SetupDependencies() {
	databases.
		GetDatabaseService().
		AddDbCreationListener(healthcheckConfigService)
}
