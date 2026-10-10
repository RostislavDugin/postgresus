package backups_core_logical

import (
	"errors"
	"fmt"
)

var (
	ErrBackupShutdown = errors.New("backup interrupted by application shutdown")
	ErrBackupTimeout  = fmt.Errorf("backup exceeded the time limit of %d hours", int(backupTimeLimit.Hours()))
)
