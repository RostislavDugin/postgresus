package users_enums

type UserRole string

const (
	UserRoleAdmin  UserRole = "ADMIN"
	UserRoleMember UserRole = "MEMBER"
)

func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleAdmin, UserRoleMember:
		return true
	default:
		return false
	}
}
