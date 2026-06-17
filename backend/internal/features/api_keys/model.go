package api_keys

import (
	"time"

	"github.com/google/uuid"

	users_enums "databasus-backend/internal/features/users/enums"
)

type ApiKey struct {
	ID              uuid.UUID            `json:"id"              gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Name            string               `json:"name"            gorm:"column:name;type:text;not null"`
	HashedToken     string               `json:"-"               gorm:"column:hashed_token;type:text;not null;uniqueIndex"`
	TokenPrefix     string               `json:"tokenPrefix"     gorm:"column:token_prefix;type:text;not null"`
	Role            users_enums.UserRole `json:"role"            gorm:"column:role;type:text;not null"`
	CreatedByUserID uuid.UUID            `json:"createdByUserId" gorm:"column:created_by_user_id;type:uuid;not null"`
	LastUsedAt      *time.Time           `json:"lastUsedAt"      gorm:"column:last_used_at"`
	ExpiresAt       *time.Time           `json:"expiresAt"       gorm:"column:expires_at"`
	RevokedAt       *time.Time           `json:"revokedAt"       gorm:"column:revoked_at"`
	CreatedAt       time.Time            `json:"createdAt"       gorm:"column:created_at"`

	Workspaces []ApiKeyWorkspace `json:"-" gorm:"foreignKey:ApiKeyID"`
}

func (ApiKey) TableName() string { return "api_keys" }

type ApiKeyWorkspace struct {
	ApiKeyID    uuid.UUID `gorm:"column:api_key_id;type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"column:workspace_id;type:uuid;primaryKey"`
}

func (ApiKeyWorkspace) TableName() string { return "api_key_workspaces" }

func (k *ApiKey) WorkspaceIDs() []uuid.UUID {
	workspaceIDs := make([]uuid.UUID, 0, len(k.Workspaces))
	for _, grant := range k.Workspaces {
		workspaceIDs = append(workspaceIDs, grant.WorkspaceID)
	}

	return workspaceIDs
}
