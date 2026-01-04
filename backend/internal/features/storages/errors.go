package storages

import "errors"

var (
	ErrInsufficientPermissionsToManageStorage = errors.New(
		"insufficient permissions to manage storage in this workspace",
	)
	ErrInsufficientPermissionsToViewStorage = errors.New(
		"insufficient permissions to view storage in this workspace",
	)
	ErrInsufficientPermissionsToViewStorages = errors.New(
		"insufficient permissions to view storages in this workspace",
	)
	ErrInsufficientPermissionsToTestStorage = errors.New(
		"insufficient permissions to test storage in this workspace",
	)
	ErrInsufficientPermissionsInSourceWorkspace = errors.New(
		"insufficient permissions to manage storage in source workspace",
	)
	ErrInsufficientPermissionsInTargetWorkspace = errors.New(
		"insufficient permissions to manage storage in target workspace",
	)
	ErrStorageDoesNotBelongToWorkspace = errors.New(
		"storage does not belong to this workspace",
	)
	ErrStorageHasAttachedDatabases = errors.New(
		"storage has attached databases and cannot be deleted",
	)
	ErrStorageHasAttachedDatabasesCannotTransfer = errors.New(
		"storage has attached databases and cannot be transferred",
	)
	ErrStorageHasOtherAttachedDatabasesCannotTransfer = errors.New(
		"storage has other attached databases and cannot be transferred",
	)
)
