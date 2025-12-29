package users_controllers

import (
	"net/http"

	user_enums "databasus-backend/internal/features/users/enums"
	user_middleware "databasus-backend/internal/features/users/middleware"
	user_models "databasus-backend/internal/features/users/models"
	users_services "databasus-backend/internal/features/users/services"

	"github.com/gin-gonic/gin"
)

type SettingsController struct {
	settingsService *users_services.SettingsService
}

func (c *SettingsController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/users/settings", c.GetUsersSettings)
	router.PUT(
		"/users/settings",
		user_middleware.RequireRole(user_enums.UserRoleAdmin),
		c.UpdateUsersSettings,
	)
}

// GetUsersSettings
// @Summary Get users settings
// @Description Get global users settings (admin only)
// @Tags settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} users_models.UsersSettings
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/settings [get]
func (c *SettingsController) GetUsersSettings(ctx *gin.Context) {
	settings, err := c.settingsService.GetSettings()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	ctx.JSON(http.StatusOK, settings)
}

// UpdateUsersSettings
// @Summary Update users settings
// @Description Update global users settings (admin only)
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body users_models.UsersSettings true "Settings update data"
// @Success 200 {object} users_models.UsersSettings
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /users/settings [put]
func (c *SettingsController) UpdateUsersSettings(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request user_models.UsersSettings
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	settings, err := c.settingsService.UpdateSettings(request, user)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, settings)
}
