package system_healthcheck

import (
	"databasus-backend/internal/features/backups/backups"
	"databasus-backend/internal/features/disk"
)

var healthcheckService = &HealthcheckService{
	disk.GetDiskService(),
	backups.GetBackupBackgroundService(),
}
var healthcheckController = &HealthcheckController{
	healthcheckService,
}

func GetHealthcheckController() *HealthcheckController {
	return healthcheckController
}
