package healthcheck_attempt

import (
	"databasus-backend/internal/config"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"log/slog"
	"time"
)

type HealthcheckAttemptBackgroundService struct {
	healthcheckConfigService   *healthcheck_config.HealthcheckConfigService
	checkDatabaseHealthUseCase *CheckDatabaseHealthUseCase
	logger                     *slog.Logger
}

func (s *HealthcheckAttemptBackgroundService) Run() {
	// first healthcheck immediately
	s.checkDatabases()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if config.IsShouldShutdown() {
			break
		}

		s.checkDatabases()
	}
}

func (s *HealthcheckAttemptBackgroundService) checkDatabases() {
	now := time.Now().UTC()

	healthcheckConfigs, err := s.healthcheckConfigService.GetDatabasesWithEnabledHealthcheck()
	if err != nil {
		s.logger.Error("failed to get databases with enabled healthcheck", "error", err)
		return
	}

	for _, healthcheckConfig := range healthcheckConfigs {
		go func(healthcheckConfig *healthcheck_config.HealthcheckConfig) {
			err := s.checkDatabaseHealthUseCase.Execute(now, healthcheckConfig)
			if err != nil {
				s.logger.Error("failed to check database health", "error", err)
			}
		}(&healthcheckConfig)
	}
}
