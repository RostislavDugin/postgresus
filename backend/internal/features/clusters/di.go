package clusters

import (
	"postgresus-backend/internal/features/backups/backups"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"postgresus-backend/internal/util/logger"
)

var clusterRepository = &ClusterRepository{}

var clusterService = &ClusterService{
	repo:                clusterRepository,
	dbService:           databases.GetDatabaseService(),
	backupService:       backups.GetBackupService(),
	backupConfigService: backups_config.GetBackupConfigService(),
	workspaceService:    workspaces_services.GetWorkspaceService(),
}

var clusterController = &ClusterController{service: clusterService}

func GetClusterController() *ClusterController { return clusterController }
func GetClusterService() *ClusterService       { return clusterService }

var clusterBackgroundService = &ClusterBackgroundService{
	service: clusterService,
	repo:    clusterRepository,
	logger:  logger.GetLogger(),
}

func GetClusterBackgroundService() *ClusterBackgroundService { return clusterBackgroundService }

func SetupDependencies() {}
