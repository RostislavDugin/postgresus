package notifiers

import (
	"net/http"
	users_middleware "postgresus-backend/internal/features/users/middleware"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotifierController struct {
	notifierService  *NotifierService
	workspaceService *workspaces_services.WorkspaceService
}

func (c *NotifierController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/notifiers", c.SaveNotifier)
	router.GET("/notifiers", c.GetNotifiers)
	router.GET("/notifiers/:id", c.GetNotifier)
	router.DELETE("/notifiers/:id", c.DeleteNotifier)
	router.POST("/notifiers/:id/test", c.SendTestNotification)
	router.POST("/notifiers/direct-test", c.SendTestNotificationDirect)
}

// SaveNotifier
// @Summary Save a notifier
// @Description Create or update a notifier
// @Tags notifiers
// @Accept json
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param request body Notifier true "Notifier data with workspaceId"
// @Success 200 {object} Notifier
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /notifiers [post]
func (c *NotifierController) SaveNotifier(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Notifier
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	if err := c.notifierService.SaveNotifier(user, request.WorkspaceID, &request); err != nil {
		if err.Error() == "insufficient permissions to manage notifier in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, request)
}

// GetNotifier
// @Summary Get a notifier by ID
// @Description Get a specific notifier by ID
// @Tags notifiers
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param id path string true "Notifier ID"
// @Success 200 {object} Notifier
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /notifiers/{id} [get]
func (c *NotifierController) GetNotifier(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	notifier, err := c.notifierService.GetNotifier(user, id)
	if err != nil {
		if err.Error() == "insufficient permissions to view notifier in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, notifier)
}

// GetNotifiers
// @Summary Get all notifiers
// @Description Get all notifiers for a workspace
// @Tags notifiers
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param workspace_id query string true "Workspace ID"
// @Success 200 {array} Notifier
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /notifiers [get]
func (c *NotifierController) GetNotifiers(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Query("workspace_id")
	if workspaceIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id query parameter is required"})
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	notifiers, err := c.notifierService.GetNotifiers(user, workspaceID)
	if err != nil {
		if err.Error() == "insufficient permissions to view notifiers in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, notifiers)
}

// DeleteNotifier
// @Summary Delete a notifier
// @Description Delete a notifier by ID
// @Tags notifiers
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param id path string true "Notifier ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /notifiers/{id} [delete]
func (c *NotifierController) DeleteNotifier(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	if err := c.notifierService.DeleteNotifier(user, id); err != nil {
		if err.Error() == "insufficient permissions to manage notifier in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "notifier deleted successfully"})
}

// SendTestNotification
// @Summary Send test notification
// @Description Send a test notification using the specified notifier
// @Tags notifiers
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param id path string true "Notifier ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /notifiers/{id}/test [post]
func (c *NotifierController) SendTestNotification(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid notifier ID"})
		return
	}

	if err := c.notifierService.SendTestNotification(user, id); err != nil {
		if err.Error() == "insufficient permissions to test notifier in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "test notification sent successfully"})
}

// SendTestNotificationDirect
// @Summary Send test notification directly
// @Description Send a test notification using a notifier object provided in the request
// @Tags notifiers
// @Accept json
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param request body Notifier true "Notifier data with workspaceId"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /notifiers/direct-test [post]
func (c *NotifierController) SendTestNotificationDirect(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Notifier
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	canView, _, err := c.workspaceService.CanUserAccessWorkspace(request.WorkspaceID, user)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !canView {
		ctx.JSON(
			http.StatusForbidden,
			gin.H{"error": "insufficient permissions to test notifier in this workspace"},
		)
		return
	}

	if err := c.notifierService.SendTestNotificationToNotifier(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "test notification sent successfully"})
}
