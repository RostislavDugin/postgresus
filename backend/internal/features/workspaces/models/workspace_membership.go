package workspaces_models

import (
	"time"

	users_enums "databasus-backend/internal/features/users/enums"

	"github.com/google/uuid"
)

type WorkspaceMembership struct {
	ID          uuid.UUID                 `json:"id"          gorm:"column:id"`
	UserID      uuid.UUID                 `json:"userId"      gorm:"column:user_id"`
	WorkspaceID uuid.UUID                 `json:"workspaceId" gorm:"column:workspace_id"`
	Role        users_enums.WorkspaceRole `json:"role"        gorm:"column:role"`
	CreatedAt   time.Time                 `json:"createdAt"   gorm:"column:created_at"`
}

func (WorkspaceMembership) TableName() string {
	return "workspace_memberships"
}
