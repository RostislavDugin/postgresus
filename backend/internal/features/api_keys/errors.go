package api_keys

import "errors"

var (
	ErrInvalidApiKey            = errors.New("invalid api key")
	ErrForbidden                = errors.New("api key not authorized for this resource")
	ErrAdminOnly                = errors.New("only administrators can manage api keys")
	ErrApiKeyNotFound           = errors.New("api key not found")
	ErrInvalidRole              = errors.New("role must be ADMIN or MEMBER")
	ErrWorkspacesRequired       = errors.New("workspaceIds are required for MEMBER api keys")
	ErrDatabaseNotFound         = errors.New("database not found")
	ErrDatabaseWithoutWorkspace = errors.New("database does not belong to a workspace")
	ErrBackupNotFound           = errors.New("backup not found")
)
