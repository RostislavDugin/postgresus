package audit_logs

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const jobName = "audit_log_cleanup"

type AuditLogBackgroundService struct {
	auditLogService *AuditLogService
	logger          *slog.Logger

	hasRun atomic.Bool
}

func (s *AuditLogBackgroundService) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	lifecycleLogger := s.logger.With("job_name", jobName)

	lifecycleLogger.InfoContext(ctx, "audit log cleanup started")

	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lifecycleLogger.InfoContext(ctx, "audit log cleanup stopped")

			return
		case <-ticker.C:
			logger := s.logger.With("job_id", uuid.New(), "job_name", jobName)
			_ = s.auditLogService.CleanOldAuditLogs(ctx, logger)
		}
	}
}
