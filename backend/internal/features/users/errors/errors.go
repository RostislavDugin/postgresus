package users_errors

import "errors"

var (
	ErrInsufficientPermissionsToInviteUsers = errors.New("insufficient permissions to invite users")
	ErrOAuthProviderNotConfigured           = errors.New("OAuth provider not configured")
)
