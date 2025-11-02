package backups

import (
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	workspaces_controllers "postgresus-backend/internal/features/workspaces/controllers"
	workspaces_testing "postgresus-backend/internal/features/workspaces/testing"

	"github.com/gin-gonic/gin"
)

func CreateTestRouter() *gin.Engine {
	return workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		backups_config.GetBackupConfigController(),
		GetBackupController(),
	)
}
