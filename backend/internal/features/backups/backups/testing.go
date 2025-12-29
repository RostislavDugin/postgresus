package backups

import (
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"

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
