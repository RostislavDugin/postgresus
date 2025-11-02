package healthcheck_config

import (
	"net/http"
	users_middleware "postgresus-backend/internal/features/users/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type HealthcheckConfigController struct {
	healthcheckConfigService *HealthcheckConfigService
}

func (c *HealthcheckConfigController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/healthcheck-config", c.SaveHealthcheckConfig)
	router.GET("/healthcheck-config/:databaseId", c.GetHealthcheckConfig)
}

// SaveHealthcheckConfig
// @Summary Save healthcheck configuration
// @Description Create or update healthcheck configuration for a database
// @Tags healthcheck-config
// @Accept json
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param config body HealthcheckConfigDTO true "Healthcheck configuration data"
// @Success 200 {object} map[string]string
// @Failure 400
// @Failure 401
// @Router /healthcheck-config [post]
func (c *HealthcheckConfigController) SaveHealthcheckConfig(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var configDTO HealthcheckConfigDTO
	if err := ctx.ShouldBindJSON(&configDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.healthcheckConfigService.Save(*user, configDTO); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Healthcheck configuration saved successfully",
	})
}

// GetHealthcheckConfig
// @Summary Get healthcheck configuration
// @Description Get healthcheck configuration for a specific database
// @Tags healthcheck-config
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param databaseId path string true "Database ID"
// @Success 200 {object} HealthcheckConfig
// @Failure 400
// @Failure 401
// @Router /healthcheck-config/{databaseId} [get]
func (c *HealthcheckConfigController) GetHealthcheckConfig(ctx *gin.Context) {
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

	config, err := c.healthcheckConfigService.GetByDatabaseID(*user, databaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, config)
}
