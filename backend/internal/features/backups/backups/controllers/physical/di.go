package backups_controllers_physical

import (
	backups_download "databasus-backend/internal/features/backups/backups/download"
	backups_services "databasus-backend/internal/features/backups/backups/services"
)

var physicalBackupController = &PhysicalBackupController{
	backups_services.GetPhysicalBackupService(),
	backups_download.GetRestoreTokenService(),
}

func GetPhysicalBackupController() *PhysicalBackupController {
	return physicalBackupController
}
