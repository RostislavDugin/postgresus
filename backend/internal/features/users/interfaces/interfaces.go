package users_interfaces

import (
	"context"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
)

type AuditLogWriter interface {
	WriteAuditLog(ctx context.Context, entry audit_logs_models.AuditEntry)
}

type EmailSender interface {
	SendEmail(to, subject, body string) error
}
