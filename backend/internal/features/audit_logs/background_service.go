package audit_logs

import (
	"databasus-backend/internal/config"
	"log/slog"
	"time"
)

type AuditLogBackgroundService struct {
	auditLogService *AuditLogService
	logger          *slog.Logger
}

func (s *AuditLogBackgroundService) Run() {
	s.logger.Info("Starting audit log cleanup background service")

	if config.IsShouldShutdown() {
		return
	}

	for {
		if config.IsShouldShutdown() {
			return
		}

		if err := s.cleanOldAuditLogs(); err != nil {
			s.logger.Error("Failed to clean old audit logs", "error", err)
		}

		time.Sleep(1 * time.Hour)
	}
}

func (s *AuditLogBackgroundService) cleanOldAuditLogs() error {
	return s.auditLogService.CleanOldAuditLogs()
}
