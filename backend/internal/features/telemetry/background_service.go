package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
)

const (
	defaultWarmupDuration  = 1 * time.Minute
	defaultSuccessInterval = 24 * time.Hour
	defaultJitter          = 10 * time.Minute
	defaultInitialBackoff  = 1 * time.Minute
	defaultMaxBackoff      = 24 * time.Hour

	jobName = "telemetry_ping"
)

type TelemetryBackgroundService struct {
	telemetryService *TelemetryService
	logger           *slog.Logger

	WarmupDuration  time.Duration
	SuccessInterval time.Duration
	Jitter          time.Duration
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration

	hasRun atomic.Bool
}

func NewTelemetryBackgroundService(
	telemetryService *TelemetryService,
	logger *slog.Logger,
) *TelemetryBackgroundService {
	return &TelemetryBackgroundService{
		telemetryService: telemetryService,
		logger:           logger,
		WarmupDuration:   defaultWarmupDuration,
		SuccessInterval:  defaultSuccessInterval,
		Jitter:           defaultJitter,
		InitialBackoff:   defaultInitialBackoff,
		MaxBackoff:       defaultMaxBackoff,
	}
}

func (s *TelemetryBackgroundService) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	if config.GetEnv().IsDisableAnonymousTelemetry {
		s.logger.InfoContext(ctx, "anonymous telemetry is disabled")
		return
	}

	if !sleep(ctx, s.WarmupDuration) {
		return
	}

	backoff := s.InitialBackoff

	for {
		logger := s.logger.With("job_id", uuid.New(), "job_name", jobName)

		err := s.telemetryService.BuildAndSend(ctx)

		var nextDelay time.Duration
		if err != nil {
			nextDelay = backoff

			logger.WarnContext(ctx, fmt.Sprintf("telemetry ping failed, retrying in %s", backoff),
				"error", err)

			backoff = min(backoff*2, s.MaxBackoff)
		} else {
			backoff = s.InitialBackoff
			nextDelay = s.SuccessInterval + jitter(s.Jitter)

			logger.InfoContext(ctx, fmt.Sprintf("telemetry ping sent, next in %s", nextDelay))
		}

		if !sleep(ctx, nextDelay) {
			return
		}
	}
}

// sleep returns false when ctx is cancelled before d elapses.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// jitter returns a random offset in [-d, +d]. With d == 0 it returns 0.
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}

	return time.Duration(rand.Int64N(int64(2*d))) - d
}
