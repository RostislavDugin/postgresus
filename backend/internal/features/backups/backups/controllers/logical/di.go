package backups_controllers_logical

import (
	backups_services "databasus-backend/internal/features/backups/backups/services"
)

var backupController = &BackupController{
	backups_services.GetBackupService(),
}

func GetBackupController() *BackupController {
	return backupController
}
