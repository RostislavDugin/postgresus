package system_healthcheck

import (
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	"databasus-backend/internal/features/disk"
	verification_agents "databasus-backend/internal/features/verification/agents"
)

var healthcheckService = &HealthcheckService{
	disk.GetDiskService(),
	backuping_logical.GetBackupsScheduler(),
	verification_agents.GetAgentService(),
}

var healthcheckController = &HealthcheckController{
	healthcheckService,
}

func GetHealthcheckController() *HealthcheckController {
	return healthcheckController
}
