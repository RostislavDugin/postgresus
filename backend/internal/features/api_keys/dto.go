package api_keys

import (
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_core "databasus-backend/internal/features/backups/backups/core"
	users_enums "databasus-backend/internal/features/users/enums"
)

type Principal struct {
	ApiKeyID     uuid.UUID
	Name         string
	Role         users_enums.UserRole
	WorkspaceIDs []uuid.UUID
}

func (p *Principal) CanAccessWorkspace(workspaceID uuid.UUID) bool {
	if p.Role == users_enums.UserRoleAdmin {
		return true
	}

	return slices.Contains(p.WorkspaceIDs, workspaceID)
}

// GetPrincipalFromContext extracts the authenticated API key principal from the gin context.
func GetPrincipalFromContext(ctx *gin.Context) (*Principal, bool) {
	value, exists := ctx.Get(PrincipalContextKey)
	if !exists {
		return nil, false
	}

	principal, isOk := value.(*Principal)

	return principal, isOk
}

type CreateApiKeyRequestDTO struct {
	Name         string               `json:"name"         binding:"required"`
	Role         users_enums.UserRole `json:"role"         binding:"required"`
	ExpiresAt    *time.Time           `json:"expiresAt"`
	WorkspaceIDs []uuid.UUID          `json:"workspaceIds"`
}

type CreateApiKeyResponseDTO struct {
	ID           uuid.UUID            `json:"id"`
	Name         string               `json:"name"`
	Role         users_enums.UserRole `json:"role"`
	Token        string               `json:"token"`
	TokenPrefix  string               `json:"tokenPrefix"`
	WorkspaceIDs []uuid.UUID          `json:"workspaceIds"`
	ExpiresAt    *time.Time           `json:"expiresAt"`
	CreatedAt    time.Time            `json:"createdAt"`
}

type ApiKeyResponseDTO struct {
	ID           uuid.UUID            `json:"id"`
	Name         string               `json:"name"`
	Role         users_enums.UserRole `json:"role"`
	TokenPrefix  string               `json:"tokenPrefix"`
	WorkspaceIDs []uuid.UUID          `json:"workspaceIds"`
	LastUsedAt   *time.Time           `json:"lastUsedAt"`
	ExpiresAt    *time.Time           `json:"expiresAt"`
	CreatedAt    time.Time            `json:"createdAt"`
}

type ListApiKeysResponseDTO struct {
	ApiKeys []*ApiKeyResponseDTO `json:"apiKeys"`
}

type TriggerBackupRequestDTO struct {
	DatabaseID     uuid.UUID `json:"databaseId"     binding:"required"`
	TimeoutSeconds *int      `json:"timeoutSeconds"`
}

type TriggerBackupResponseDTO struct {
	BackupID                  uuid.UUID                              `json:"backupId"`
	Status                    backups_core.BackupStatus              `json:"status,omitempty"`
	FailMessage               *string                                `json:"failMessage,omitempty"`
	RestoreVerificationStatus backups_core.RestoreVerificationStatus `json:"restoreVerificationStatus,omitempty"`
	Error                     *string                                `json:"error,omitempty"`
}
