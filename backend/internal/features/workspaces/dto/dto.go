package workspaces_dto

import (
	"time"

	users_enums "databasus-backend/internal/features/users/enums"

	"github.com/google/uuid"
)

type AddMemberStatus string

const (
	AddStatusInvited AddMemberStatus = "INVITED"
	AddStatusAdded   AddMemberStatus = "ADDED"
)

// Workspace DTOs
type CreateWorkspaceRequestDTO struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

type WorkspaceResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`

	// User's role in this workspace (populated when fetching for specific user)
	UserRole *users_enums.WorkspaceRole `json:"userRole,omitempty"`
}

type ListWorkspacesResponseDTO struct {
	Workspaces []WorkspaceResponseDTO `json:"workspaces"`
}

// Membership DTOs
type AddMemberRequestDTO struct {
	Email string                    `json:"email" binding:"required,email"`
	Role  users_enums.WorkspaceRole `json:"role"  binding:"required"`
}

type AddMemberResponseDTO struct {
	Status AddMemberStatus `json:"status"`
}

type ChangeMemberRoleRequestDTO struct {
	Role users_enums.WorkspaceRole `json:"role" binding:"required"`
}

type TransferOwnershipRequestDTO struct {
	NewOwnerEmail string `json:"newOwnerEmail" binding:"required"`
}

type WorkspaceMemberResponseDTO struct {
	ID        uuid.UUID                 `json:"id"`
	UserID    uuid.UUID                 `json:"userId"`
	Email     string                    `json:"email"` // Populated from user join
	Name      string                    `json:"name"`  // Populated from user join
	Role      users_enums.WorkspaceRole `json:"role"`
	CreatedAt time.Time                 `json:"createdAt"`
}

type GetMembersResponseDTO struct {
	Members []WorkspaceMemberResponseDTO `json:"members"`
}
