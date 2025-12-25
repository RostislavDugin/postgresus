package workspaces_controllers

import (
	"net/http"

	audit_logs "databasus-backend/internal/features/audit_logs"
	users_middleware "databasus-backend/internal/features/users/middleware"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WorkspaceController struct {
	workspaceService *workspaces_services.WorkspaceService
}

func (c *WorkspaceController) RegisterRoutes(router *gin.RouterGroup) {
	workspaceRoutes := router.Group("/workspaces")

	workspaceRoutes.POST("", c.CreateWorkspace)
	workspaceRoutes.GET("", c.GetWorkspaces)
	workspaceRoutes.GET("/:id", c.GetWorkspace)
	workspaceRoutes.PUT("/:id", c.UpdateWorkspace)
	workspaceRoutes.DELETE("/:id", c.DeleteWorkspace)
	workspaceRoutes.GET("/:id/audit-logs", c.GetWorkspaceAuditLogs)
}

// CreateWorkspace
// @Summary Create a new workspace
// @Description Create a new workspace with default settings
// @Tags workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body workspaces_dto.CreateWorkspaceRequestDTO true "Workspace creation data"
// @Success 200 {object} workspaces_dto.WorkspaceResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces [post]
func (c *WorkspaceController) CreateWorkspace(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request workspaces_dto.CreateWorkspaceRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.workspaceService.CreateWorkspace(&request, user)
	if err != nil {
		if err.Error() == "insufficient permissions to create workspaces" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetWorkspaces
// @Summary List user's workspaces
// @Description Get list of workspaces the user is a member of
// @Tags workspaces
// @Produce json
// @Security BearerAuth
// @Success 200 {object} workspaces_dto.ListWorkspacesResponseDTO
// @Failure 401 {object} map[string]string
// @Router /workspaces [get]
func (c *WorkspaceController) GetWorkspaces(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	response, err := c.workspaceService.GetUserWorkspaces(user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workspaces"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetWorkspace
// @Summary Get workspace details
// @Description Get detailed information about a specific workspace
// @Tags workspaces
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Success 200 {object} workspaces_models.Workspace
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/{id} [get]
func (c *WorkspaceController) GetWorkspace(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	workspace, err := c.workspaceService.GetWorkspace(workspaceID, user)
	if err != nil {
		if err.Error() == "insufficient permissions to view workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, workspace)
}

// UpdateWorkspace
// @Summary Update workspace settings
// @Description Update workspace configuration and settings
// @Tags workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Param request body workspaces_models.Workspace true "Workspace update data"
// @Success 200 {object} workspaces_models.Workspace
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/{id} [put]
func (c *WorkspaceController) UpdateWorkspace(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	var workspace workspaces_models.Workspace
	if err := ctx.ShouldBindJSON(&workspace); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	updatedWorkspace, err := c.workspaceService.UpdateWorkspace(workspaceID, &workspace, user)
	if err != nil {
		if err.Error() == "insufficient permissions to update workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updatedWorkspace)
}

// DeleteWorkspace
// @Summary Delete workspace
// @Description Delete a workspace (owner only)
// @Tags workspaces
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/{id} [delete]
func (c *WorkspaceController) DeleteWorkspace(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	if err := c.workspaceService.DeleteWorkspace(workspaceID, user); err != nil {
		if err.Error() == "only workspace owner or admin can delete workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Workspace deleted successfully"})
}

// GetWorkspaceAuditLogs
// @Summary Get workspace audit logs
// @Description Retrieve audit logs for a specific workspace (member access required)
// @Tags workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} audit_logs.GetAuditLogsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/{id}/audit-logs [get]
func (c *WorkspaceController) GetWorkspaceAuditLogs(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	request := &audit_logs.GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.workspaceService.GetWorkspaceAuditLogs(workspaceID, user, request)
	if err != nil {
		if err.Error() == "insufficient permissions to view workspace audit logs" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
