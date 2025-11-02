package audit_logs

import (
	"time"

	"github.com/google/uuid"
)

type GetAuditLogsRequest struct {
	Limit      int        `form:"limit"      json:"limit"`
	Offset     int        `form:"offset"     json:"offset"`
	BeforeDate *time.Time `form:"beforeDate" json:"beforeDate"`
}

type GetAuditLogsResponse struct {
	AuditLogs []*AuditLogDTO `json:"auditLogs"`
	Total     int64          `json:"total"`
	Limit     int            `json:"limit"`
	Offset    int            `json:"offset"`
}

type AuditLogDTO struct {
	ID            uuid.UUID  `json:"id"            gorm:"column:id"`
	UserID        *uuid.UUID `json:"userId"        gorm:"column:user_id"`
	WorkspaceID   *uuid.UUID `json:"workspaceId"   gorm:"column:workspace_id"`
	Message       string     `json:"message"       gorm:"column:message"`
	CreatedAt     time.Time  `json:"createdAt"     gorm:"column:created_at"`
	UserEmail     *string    `json:"userEmail"     gorm:"column:user_email"`
	UserName      *string    `json:"userName"      gorm:"column:user_name"`
	WorkspaceName *string    `json:"workspaceName" gorm:"column:workspace_name"`
}
