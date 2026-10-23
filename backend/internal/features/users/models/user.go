package users_models

import (
	"time"

	"github.com/google/uuid"

	users_enums "databasus-backend/internal/features/users/enums"
)

// MinPasswordLength applies to every way a password is set. The request DTOs
// repeat it as a literal min=8, because a struct tag cannot reference a constant.
const MinPasswordLength = 8

type User struct {
	ID                   uuid.UUID              `json:"id"`
	Email                string                 `json:"email"`
	Name                 string                 `json:"name"`
	HashedPassword       *string                `json:"-"         gorm:"column:hashed_password"`
	PasswordCreationTime time.Time              `json:"-"         gorm:"column:password_creation_time"`
	Role                 users_enums.UserRole   `json:"role"`
	Status               users_enums.UserStatus `json:"status"`
	GitHubOAuthID        *string                `json:"-"         gorm:"column:github_oauth_id"`
	GoogleOAuthID        *string                `json:"-"         gorm:"column:google_oauth_id"`
	OIDCIssuer           *string                `json:"-"         gorm:"column:oidc_issuer"`
	OIDCSubject          *string                `json:"-"         gorm:"column:oidc_subject"`
	IsRootAdmin          bool                   `json:"-"         gorm:"column:is_root_admin"`
	CreatedAt            time.Time              `json:"createdAt"`
}

func (User) TableName() string {
	return "users"
}

// Permission methods
func (u *User) CanInviteUsers(settings *UsersSettings) bool {
	if u.Role == users_enums.UserRoleAdmin {
		return true
	}

	return u.Role == users_enums.UserRoleMember && settings.IsAllowMemberInvitations
}

func (u *User) CanManageUsers() bool {
	return u.Role == users_enums.UserRoleAdmin
}

func (u *User) CanUpdateSettings() bool {
	return u.Role == users_enums.UserRoleAdmin
}

func (u *User) CanCreateWorkspaces(settings *UsersSettings) bool {
	if u.Role == users_enums.UserRoleAdmin {
		return true
	}
	return u.Role == users_enums.UserRoleMember && settings.IsMemberAllowedToCreateWorkspaces
}

func (u *User) IsActiveUser() bool {
	return u.Status == users_enums.UserStatusActive
}

func (u *User) HasPassword() bool {
	return u.HashedPassword != nil && *u.HashedPassword != ""
}
