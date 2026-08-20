package healthcheck_attempt

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
)

const jobName = "healthcheck_attempt_scan"

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
	s.checkDatabases(ctx)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	lifecycleLogger := s.logger.With("job_name", jobName)

	lifecycleLogger.InfoContext(ctx, "database healthcheck started")

	for {
		select {
		case <-ctx.Done():
			lifecycleLogger.InfoContext(ctx, "database healthcheck stopped")

			return
		case <-ticker.C:
			s.checkDatabases(ctx)
		}
	}
}

func (s *HealthcheckAttemptBackgroundService) checkDatabases(ctx context.Context) {
	now := time.Now().UTC()
	logger := s.logger.With("job_id", uuid.New(), "job_name", jobName)

	healthcheckConfigs, err := s.healthcheckConfigService.GetDatabasesWithEnabledHealthcheck()
	if err != nil {
		logger.ErrorContext(ctx, "failed to get databases with enabled healthcheck", "error", err)

		return
	}

	logger.DebugContext(ctx, fmt.Sprintf("probing %d databases with healthcheck enabled",
		len(healthcheckConfigs)))

	for _, healthcheckConfig := range healthcheckConfigs {
		go func() {
			databaseLogger := logger.With("database_id", healthcheckConfig.DatabaseID)

			if err := s.checkDatabaseHealthUseCase.Execute(ctx, databaseLogger, now, &healthcheckConfig); err != nil {
				databaseLogger.ErrorContext(ctx, "failed to check database health", "error", err)
			}
		}()
	}
}
