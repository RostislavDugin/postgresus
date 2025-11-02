package users_enums

type UserStatus string

const (
	UserStatusInvited  UserStatus = "INVITED"
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
)
