package users_errors

import "errors"

var (
	ErrInsufficientPermissionsToInviteUsers = errors.New("insufficient permissions to invite users")
)
