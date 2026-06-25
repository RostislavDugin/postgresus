package download_token

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

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

	s.logger.Info("Starting download token cleanup background service")

	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.downloadTokenService.CleanExpiredTokens(); err != nil {
				s.logger.Error("Failed to clean expired download tokens", "error", err)
			}
		}
	}
}
