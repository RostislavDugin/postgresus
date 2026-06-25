package backups_services

import "errors"

var (
	ErrBackupNotStarted   = errors.New("backup could not be started")
	ErrBackupWaitTimeout  = errors.New("timed out waiting for backup to finish")
	ErrAgentManagedBackup = errors.New(
		"physical (agent-managed) databases cannot be backed up via the synchronous logical trigger; they back up through their agent",
	)
)
