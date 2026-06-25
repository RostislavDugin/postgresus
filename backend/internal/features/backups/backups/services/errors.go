package backups_services

import "errors"

var (
	ErrBackupNotStarted   = errors.New("backup could not be started")
	ErrBackupWaitTimeout  = errors.New("timed out waiting for backup to finish")
	ErrAgentManagedBackup = errors.New(
		"agent-managed (WAL_V1) databases cannot be backed up via the synchronous trigger; they back up through their agent",
	)
)
