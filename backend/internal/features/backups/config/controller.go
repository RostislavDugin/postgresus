package backups_config

import (
	"errors"
	"net/http"

	users_middleware "databasus-backend/internal/features/users/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BackupConfigController struct {
	backupConfigService *BackupConfigService
}

func (c *BackupConfigController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/backup-configs/:id", c.GetBackupConfigByID)
	router.GET("/backup-configs/database/:databaseId", c.GetBackupConfigsByDatabaseID)
	router.POST("/backup-configs", c.CreateBackupConfig)
	router.PUT("/backup-configs/:id", c.UpdateBackupConfig)
	router.DELETE("/backup-configs/:id", c.DeleteBackupConfig)
	router.GET("/backup-configs/database/:id/plan", c.GetDatabasePlan)
	router.GET("/backup-configs/storage/:id/is-using", c.IsStorageUsing)
	router.GET("/backup-configs/storage/:id/databases-count", c.CountDatabasesForStorage)
	router.POST("/backup-configs/database/:id/transfer", c.TransferDatabase)
}

// GetBackupConfigByID
// @Summary Get backup configuration by ID
// @Description Get a specific backup configuration by its ID
// @Tags backup-configs
// @Produce json
// @Param id path string true "Backup Config ID"
// @Success 200 {object} BackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backup-configs/{id} [get]
func (c *BackupConfigController) GetBackupConfigByID(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup config ID"})
		return
	}

	backupConfig, err := c.backupConfigService.GetBackupConfigByID(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if backupConfig == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "backup configuration not found"})
		return
	}

	// Verify user has access to the database
	_, err = c.backupConfigService.GetDatabasePlan(user, backupConfig.DatabaseID)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	ctx.JSON(http.StatusOK, backupConfig)
}

// GetBackupConfigsByDatabaseID
// @Summary Get backup configurations by database ID
// @Description Get all backup configurations for a specific database
// @Tags backup-configs
// @Produce json
// @Param databaseId path string true "Database ID"
// @Success 200 {array} BackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /backup-configs/database/{databaseId} [get]
func (c *BackupConfigController) GetBackupConfigsByDatabaseID(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("databaseId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	backupConfigs, err := c.backupConfigService.GetBackupConfigsByDatabaseIDWithAuth(user, databaseID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, backupConfigs)
}

// CreateBackupConfig
// @Summary Create a new backup configuration
// @Description Create a new backup configuration for a database
// @Tags backup-configs
// @Accept json
// @Produce json
// @Param request body BackupConfig true "Backup configuration data"
// @Success 201 {object} BackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /backup-configs [post]
func (c *BackupConfigController) CreateBackupConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var requestDTO BackupConfig
	if err := ctx.ShouldBindJSON(&requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Clear StorageID to rely on full Storage object
	requestDTO.StorageID = nil

	savedConfig, err := c.backupConfigService.CreateBackupConfig(user, &requestDTO)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, savedConfig)
}

// UpdateBackupConfig
// @Summary Update a backup configuration
// @Description Update an existing backup configuration
// @Tags backup-configs
// @Accept json
// @Produce json
// @Param id path string true "Backup Config ID"
// @Param request body BackupConfig true "Backup configuration data"
// @Success 200 {object} BackupConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backup-configs/{id} [put]
func (c *BackupConfigController) UpdateBackupConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup config ID"})
		return
	}

	var requestDTO BackupConfig
	if err := ctx.ShouldBindJSON(&requestDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Clear StorageID to rely on full Storage object
	requestDTO.StorageID = nil

	savedConfig, err := c.backupConfigService.UpdateBackupConfig(user, id, &requestDTO)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, savedConfig)
}

// DeleteBackupConfig
// @Summary Delete a backup configuration
// @Description Delete an existing backup configuration
// @Tags backup-configs
// @Param id path string true "Backup Config ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backup-configs/{id} [delete]
func (c *BackupConfigController) DeleteBackupConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup config ID"})
		return
	}

	if err := c.backupConfigService.DeleteBackupConfig(user, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetDatabasePlan
// @Summary Get database plan by database ID
// @Description Get the plan limits for a specific database (max backup size, max total size, max storage period)
// @Tags backup-configs
// @Produce json
// @Param id path string true "Database ID"
// @Success 200 {object} plans.DatabasePlan
// @Failure 400 {object} map[string]string "Invalid database ID"
// @Failure 401 {object} map[string]string "User not authenticated"
// @Failure 404 {object} map[string]string "Database not found or access denied"
// @Router /backup-configs/database/{id}/plan [get]
func (c *BackupConfigController) GetDatabasePlan(ctx *gin.Context) {
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

	plan, err := c.backupConfigService.GetDatabasePlan(user, id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "database plan not found"})
		return
	}

	ctx.JSON(http.StatusOK, plan)
}

// IsStorageUsing
// @Summary Check if storage is being used
// @Description Check if a storage is currently being used by any backup configuration
// @Tags backup-configs
// @Produce json
// @Param id path string true "Storage ID"
// @Success 200 {object} map[string]bool
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /backup-configs/storage/{id}/is-using [get]
func (c *BackupConfigController) IsStorageUsing(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage ID"})
		return
	}

	isUsing, err := c.backupConfigService.IsStorageUsing(user, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"isUsing": isUsing})
}

// CountDatabasesForStorage
// @Summary Count databases using a storage
// @Description Get the count of databases that are using a specific storage
// @Tags backup-configs
// @Produce json
// @Param id path string true "Storage ID"
// @Success 200 {object} map[string]int
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /backup-configs/storage/{id}/databases-count [get]
func (c *BackupConfigController) CountDatabasesForStorage(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage ID"})
		return
	}

	count, err := c.backupConfigService.CountDatabasesForStorage(user, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"count": count})
}

// TransferDatabase
// @Summary Transfer database to another workspace
// @Description Transfer a database from one workspace to another. Can transfer to a new storage or transfer with the existing storage. Can also specify target notifiers from the target workspace.
// @Tags backup-configs
// @Accept json
// @Produce json
// @Param id path string true "Database ID"
// @Param request body TransferDatabaseRequest true "Transfer request with targetWorkspaceId, storage options (targetStorageId or isTransferWithStorage), and optional targetNotifierIds"
// @Success 200 {object} map[string]string "Database transferred successfully"
// @Failure 400 {object} map[string]string "Invalid request, target storage/notifier not in target workspace, or transfer failed"
// @Failure 401 {object} map[string]string "User not authenticated"
// @Failure 403 {object} map[string]string "Insufficient permissions"
// @Router /backup-configs/database/{id}/transfer [post]
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
