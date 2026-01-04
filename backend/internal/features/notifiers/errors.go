package notifiers

import "errors"

var (
	ErrInsufficientPermissionsToManageNotifier = errors.New(
		"insufficient permissions to manage notifier in this workspace",
	)
	ErrInsufficientPermissionsToViewNotifier = errors.New(
		"insufficient permissions to view notifier in this workspace",
	)
	ErrInsufficientPermissionsToViewNotifiers = errors.New(
		"insufficient permissions to view notifiers in this workspace",
	)
	ErrInsufficientPermissionsToTestNotifier = errors.New(
		"insufficient permissions to test notifier in this workspace",
	)
	ErrNotifierDoesNotBelongToWorkspace = errors.New(
		"notifier does not belong to this workspace",
	)
	ErrInsufficientPermissionsInSourceWorkspace = errors.New(
		"insufficient permissions to manage notifier in source workspace",
	)
	ErrInsufficientPermissionsInTargetWorkspace = errors.New(
		"insufficient permissions to manage notifier in target workspace",
	)
	ErrNotifierHasAttachedDatabases = errors.New(
		"notifier has attached databases and cannot be deleted",
	)
	ErrNotifierHasAttachedDatabasesCannotTransfer = errors.New(
		"notifier has attached databases and cannot be transferred",
	)
	ErrNotifierHasOtherAttachedDatabasesCannotTransfer = errors.New(
		"notifier has other attached databases and cannot be transferred",
	)
)
