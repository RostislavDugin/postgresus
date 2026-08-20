package download_token

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const jobName = "download_token_cleanup"

type BackgroundService struct {
	downloadTokenService *Service
	logger               *slog.Logger

	hasRun atomic.Bool
}

func NewBackgroundService(downloadTokenService *Service, logger *slog.Logger) *BackgroundService {
	return &BackgroundService{
		downloadTokenService: downloadTokenService,
		logger:               logger,
	}
}

func (s *BackgroundService) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	lifecycleLogger := s.logger.With("job_name", jobName)

	lifecycleLogger.InfoContext(ctx, "download token cleanup started")

	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lifecycleLogger.InfoContext(ctx, "download token cleanup stopped")

			return
		case <-ticker.C:
			logger := s.logger.With("job_id", uuid.New(), "job_name", jobName)

			deletedCount, err := s.downloadTokenService.CleanExpiredTokens(ctx)
			if err != nil {
				logger.ErrorContext(ctx, "failed to clean expired download tokens", "error", err)

				continue
			}

			logger.DebugContext(ctx, fmt.Sprintf("deleted %d expired download tokens", deletedCount))
		}
	}
}
