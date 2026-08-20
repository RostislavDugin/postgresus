package verification_config

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_middleware "databasus-backend/internal/features/users/middleware"
)

type VerificationConfigController struct {
	verificationConfigService *VerificationConfigService
}

func (c *VerificationConfigController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/verification-config/:databaseId", c.GetByDatabaseID)
	router.PUT("/verification-config/:databaseId", c.Save)
}

// GetByDatabaseID
// @Summary Get backup verification configuration
// @Description Get backup verification configuration for a specific database. Lazily creates a disabled default on first read.
// @Tags verification-config
// @Produce json
// @Security BearerAuth
// @Param databaseId path string true "Database ID"
// @Success 200 {object} BackupVerificationConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /verification-config/{databaseId} [get]
func (c *VerificationConfigController) GetByDatabaseID(ctx *gin.Context) {
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

	config, err := c.verificationConfigService.GetByDatabaseID(ctx.Request.Context(), user, databaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, config)
}

// Save
// @Summary Save backup verification configuration
// @Description Create or update backup verification configuration for a database. Verification cannot be enabled for WAL-based PostgreSQL databases.
// @Tags verification-config
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param databaseId path string true "Database ID"
// @Param request body SaveBackupVerificationConfigDTO true "Verification configuration"
// @Success 200 {object} BackupVerificationConfig
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /verification-config/{databaseId} [put]
func (c *VerificationConfigController) Save(ctx *gin.Context) {
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

	var req SaveBackupVerificationConfigDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := c.verificationConfigService.Save(ctx.Request.Context(), user, databaseID, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, config)
}
