package workspaces_errors

import "errors"

var (
	// Workspace errors
	ErrInsufficientPermissionsToCreateWorkspaces = errors.New(
		"insufficient permissions to create workspaces",
	)
	ErrInsufficientPermissionsToViewWorkspace = errors.New(
		"insufficient permissions to view workspace",
	)
	ErrInsufficientPermissionsToUpdateWorkspace = errors.New(
		"insufficient permissions to update workspace",
	)
	ErrInsufficientPermissionsToViewWorkspaceAuditLogs = errors.New(
		"insufficient permissions to view workspace audit logs",
	)
	ErrOnlyOwnerOrAdminCanDeleteWorkspace = errors.New(
		"only workspace owner or admin can delete workspace",
	)

	// Membership errors
	ErrInsufficientPermissionsToViewMembers = errors.New(
		"insufficient permissions to view workspace members",
	)
	ErrInsufficientPermissionsToManageMembers = errors.New(
		"insufficient permissions to manage members",
	)
	ErrInsufficientPermissionsToRemoveMembers = errors.New(
		"insufficient permissions to remove members",
	)
	ErrInsufficientPermissionsToInviteUsers = errors.New(
		"insufficient permissions to invite users",
	)
	ErrOnlyOwnerCanAddManageAdmins = errors.New(
		"only workspace owner can add/manage admins",
	)
	ErrOnlyOwnerCanRemoveAdmins             = errors.New("only workspace owner can remove admins")
	ErrOnlyOwnerOrAdminCanTransferOwnership = errors.New(
		"only workspace owner or admin can transfer ownership",
	)
	ErrUserAlreadyMember = errors.New(
		"user is already a member of this workspace",
	)
	ErrCannotChangeOwnRole        = errors.New("cannot change your own role")
	ErrUserNotMemberOfWorkspace   = errors.New("user is not a member of this workspace")
	ErrCannotChangeOwnerRole      = errors.New("cannot change owner role")
	ErrUserNotFound               = errors.New("user not found")
	ErrCannotRemoveWorkspaceOwner = errors.New(
		"cannot remove workspace owner, transfer ownership first",
	)
	ErrNewOwnerNotFound        = errors.New("new owner not found")
	ErrNewOwnerMustBeMember    = errors.New("new owner must be a workspace member")
	ErrNoCurrentWorkspaceOwner = errors.New("no current workspace owner found")
)
