package healthcheck_attempt

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
)

type HealthcheckAttemptBackgroundService struct {
	healthcheckConfigService   *healthcheck_config.HealthcheckConfigService
	checkDatabaseHealthUseCase *CheckDatabaseHealthUseCase
	logger                     *slog.Logger

	hasRun atomic.Bool
}

func (s *HealthcheckAttemptBackgroundService) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	// first healthcheck immediately
	s.checkDatabases()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkDatabases()
		}
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
		go func() {
			err := s.checkDatabaseHealthUseCase.Execute(now, &healthcheckConfig)
			if err != nil {
				s.logger.Error(
					"failed to check database health",
					"database_id", healthcheckConfig.DatabaseID,
					"error", err,
				)
			}
		}()
	}
}
