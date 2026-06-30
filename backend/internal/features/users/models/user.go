package users_models

import (
	"time"

	"github.com/google/uuid"

	users_enums "databasus-backend/internal/features/users/enums"
)

type User struct {
	ID                   uuid.UUID              `json:"id"`
	Email                string                 `json:"email"`
	Name                 string                 `json:"name"`
	HashedPassword       *string                `json:"-"         gorm:"column:hashed_password"`
	PasswordCreationTime time.Time              `json:"-"         gorm:"column:password_creation_time"`
	Role                 users_enums.UserRole   `json:"role"`
	Status               users_enums.UserStatus `json:"status"`
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
