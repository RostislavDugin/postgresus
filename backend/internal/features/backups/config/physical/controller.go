package backups_config_physical

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_middleware "databasus-backend/internal/features/users/middleware"
)

type BackupConfigController struct {
	backupConfigService *BackupConfigService
}

func (c *BackupConfigController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/backup-configs/physical/save", c.SaveBackupConfig)
	router.GET("/backup-configs/physical/database/:id", c.GetBackupConfigByDbID)
	router.POST("/backup-configs/physical/database/:id/transfer", c.TransferDatabase)
}

// SaveBackupConfig
// @Summary Save physical backup configuration
// @Description Save or update physical backup configuration for a database. Encryption can be set to NONE (no encryption) or ENCRYPTED (AES-256-GCM encryption).
// @Tags backup-configs-physical
// @Accept json
// @Produce json
// @Param request body PhysicalBackupConfig true "Physical backup configuration"
// @Success 200 {object} PhysicalBackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /backup-configs/physical/save [post]
func (c *BackupConfigController) SaveBackupConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var requestDTO PhysicalBackupConfig
	if err := ctx.ShouldBindJSON(&requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requestDTO.StorageID = nil

	savedConfig, err := c.backupConfigService.SaveBackupConfigWithAuth(user, &requestDTO)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, savedConfig)
}

// GetBackupConfigByDbID
// @Summary Get physical backup configuration by database ID
// @Description Get physical backup configuration for a specific database
// @Tags backup-configs-physical
// @Produce json
// @Param id path string true "Database ID"
// @Success 200 {object} PhysicalBackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backup-configs/physical/database/{id} [get]
func (c *BackupConfigController) GetBackupConfigByDbID(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	backupConfig, err := c.backupConfigService.GetBackupConfigByDbIdWithAuth(user, id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "backup configuration not found"})
		return
	}

	ctx.JSON(http.StatusOK, backupConfig)
}

// TransferDatabase
// @Summary Transfer a database with physical backup config to another workspace
// @Description Transfer a database from one workspace to another, optionally moving its storage and notifiers.
// @Tags backup-configs-physical
// @Accept json
// @Produce json
// @Param id path string true "Database ID"
// @Param request body TransferDatabaseRequest true "Transfer request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /backup-configs/physical/database/{id}/transfer [post]
func (c *BackupConfigController) TransferDatabase(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request TransferDatabaseRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.TargetWorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "targetWorkspaceId is required"})
		return
	}

	if err := c.backupConfigService.TransferDatabaseToWorkspace(user, id, &request); err != nil {
		if errors.Is(err, ErrInsufficientPermissionsInSourceWorkspace) ||
			errors.Is(err, ErrInsufficientPermissionsInTargetWorkspace) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "database transferred successfully"})
}
