package healthcheck_attempt

import (
	"databasus-backend/internal/features/databases"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/notifiers"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var healthcheckAttemptRepository = &HealthcheckAttemptRepository{}
var healthcheckAttemptService = &HealthcheckAttemptService{
	healthcheckAttemptRepository,
	databases.GetDatabaseService(),
	workspaces_services.GetWorkspaceService(),
}

var checkDatabaseHealthUseCase = &CheckDatabaseHealthUseCase{
	healthcheckAttemptRepository,
	notifiers.GetNotifierService(),
	databases.GetDatabaseService(),
}

var healthcheckAttemptBackgroundService = &HealthcheckAttemptBackgroundService{
	healthcheck_config.GetHealthcheckConfigService(),
	checkDatabaseHealthUseCase,
	logger.GetLogger(),
}
var healthcheckAttemptController = &HealthcheckAttemptController{
	healthcheckAttemptService,
}

func GetHealthcheckAttemptRepository() *HealthcheckAttemptRepository {
	return healthcheckAttemptRepository
}

func GetHealthcheckAttemptService() *HealthcheckAttemptService {
	return healthcheckAttemptService
}

func GetHealthcheckAttemptBackgroundService() *HealthcheckAttemptBackgroundService {
	return healthcheckAttemptBackgroundService
}

func GetHealthcheckAttemptController() *HealthcheckAttemptController {
	return healthcheckAttemptController
}
