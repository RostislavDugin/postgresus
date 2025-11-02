package users_interfaces

import (
	"github.com/google/uuid"
)

type AuditLogWriter interface {
	WriteAuditLog(message string, userID *uuid.UUID, workspaceID *uuid.UUID)
}
