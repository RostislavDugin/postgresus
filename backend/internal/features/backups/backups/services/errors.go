package backups_services

import "errors"

var (
	ErrBackupNotStarted  = errors.New("backup could not be started")
	ErrBackupWaitTimeout = errors.New("timed out waiting for backup to finish")
)
