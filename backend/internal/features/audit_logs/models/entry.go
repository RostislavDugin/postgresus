// Package audit_logs_models sits below audit_logs because users_interfaces declares the writer
// seam while audit_logs imports users_services to wire it up, so the DTO cannot live in either.
package audit_logs_models

import "github.com/google/uuid"

type AuditEntry struct {
	Message     string
	UserID      *uuid.UUID
	WorkspaceID *uuid.UUID
}
