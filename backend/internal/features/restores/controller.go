package restores

import (
	"net/http"
	users_middleware "postgresus-backend/internal/features/users/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RestoreController struct {
	restoreService *RestoreService
}

func (c *RestoreController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/restores/:backupId", c.GetRestores)
	router.POST("/restores/:backupId/restore", c.RestoreBackup)
}

// GetRestores
// @Summary Get restores for a backup
// @Description Get all restores for a specific backup
// @Tags restores
// @Produce json
// @Param backupId path string true "Backup ID"
// @Success 200 {array} models.Restore
// @Failure 400
// @Failure 401
// @Router /restores/{backupId} [get]
func (c *RestoreController) GetRestores(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	restores, err := c.restoreService.GetRestores(user, backupID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, restores)
}

// RestoreBackup
// @Summary Restore a backup
// @Description Start a restore process for a specific backup
// @Tags restores
// @Param backupId path string true "Backup ID"
// @Success 200 {object} map[string]string
// @Failure 400
// @Failure 401
// @Router /restores/{backupId}/restore [post]
func (c *RestoreController) RestoreBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	var requestDTO RestoreBackupRequest
	if err := ctx.ShouldBindJSON(&requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.restoreService.RestoreBackupWithAuth(user, backupID, requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "restore started successfully"})
}
