package users_dto

import (
	"time"

	users_enums "postgresus-backend/internal/features/users/enums"

	"github.com/google/uuid"
)

type SignUpRequestDTO struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"     binding:"required"`
}

type SignInRequestDTO struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SignInResponseDTO struct {
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
	Token  string    `json:"token"`
}

type SetAdminPasswordRequestDTO struct {
	Password string `json:"password" binding:"required,min=8"`
}

type IsAdminHasPasswordResponseDTO struct {
	HasPassword bool `json:"hasPassword"`
}

type ChangePasswordRequestDTO struct {
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type UpdateUserInfoRequestDTO struct {
	Name  *string `json:"name"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type InviteUserRequestDTO struct {
	Email                 string                     `json:"email"                 binding:"required,email"`
	IntendedWorkspaceID   *uuid.UUID                 `json:"intendedWorkspaceId"`
	IntendedWorkspaceRole *users_enums.WorkspaceRole `json:"intendedWorkspaceRole"`
}

type InviteUserResponseDTO struct {
	ID                    uuid.UUID                  `json:"id"`
	Email                 string                     `json:"email"`
	IntendedWorkspaceID   *uuid.UUID                 `json:"intendedWorkspaceId"`
	IntendedWorkspaceRole *users_enums.WorkspaceRole `json:"intendedWorkspaceRole"`
	CreatedAt             time.Time                  `json:"createdAt"`
}

type UserProfileResponseDTO struct {
	ID        uuid.UUID            `json:"id"`
	Email     string               `json:"email"`
	Name      string               `json:"name"`
	Role      users_enums.UserRole `json:"role"`
	IsActive  bool                 `json:"isActive"`
	CreatedAt time.Time            `json:"createdAt"`
}

type ListUsersResponseDTO struct {
	Users []UserProfileResponseDTO `json:"users"`
	Total int64                    `json:"total"`
}

type ChangeUserRoleRequestDTO struct {
	Role users_enums.UserRole `json:"role" binding:"required"`
}

type ListUsersRequestDTO struct {
	Limit      int        `form:"limit"      json:"limit"`
	Offset     int        `form:"offset"     json:"offset"`
	BeforeDate *time.Time `form:"beforeDate" json:"beforeDate"`
	Query      string     `form:"query"      json:"query"`
}

type OAuthCallbackRequestDTO struct {
	Code        string `json:"code"        binding:"required"`
	RedirectUri string `json:"redirectUri" binding:"required"`
}

type OAuthCallbackResponseDTO struct {
	UserID    uuid.UUID `json:"userId"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	IsNewUser bool      `json:"isNewUser"`
}
