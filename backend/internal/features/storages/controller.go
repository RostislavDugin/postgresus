package storages

import (
	users_middleware "databasus-backend/internal/features/users/middleware"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type StorageController struct {
	storageService   *StorageService
	workspaceService *workspaces_services.WorkspaceService
}

func (c *StorageController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/storages", c.SaveStorage)
	router.GET("/storages", c.GetStorages)
	router.GET("/storages/:id", c.GetStorage)
	router.DELETE("/storages/:id", c.DeleteStorage)
	router.POST("/storages/:id/test", c.TestStorageConnection)
	router.POST("/storages/direct-test", c.TestStorageConnectionDirect)
}

// SaveStorage
// @Summary Save a storage
// @Description Create or update a storage
// @Tags storages
// @Accept json
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param request body Storage true "Storage data with workspaceId"
// @Success 200 {object} Storage
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /storages [post]
func (c *StorageController) SaveStorage(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Storage
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == uuid.Nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	if err := c.storageService.SaveStorage(user, request.WorkspaceID, &request); err != nil {
		if err.Error() == "insufficient permissions to manage storage in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, request)
}

// GetStorage
// @Summary Get a storage by ID
// @Description Get a specific storage by ID
// @Tags storages
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param id path string true "Storage ID"
// @Success 200 {object} Storage
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /storages/{id} [get]
func (c *StorageController) GetStorage(ctx *gin.Context) {
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

	storage, err := c.storageService.GetStorage(user, id)
	if err != nil {
		if err.Error() == "insufficient permissions to view storage in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, storage)
}

// GetStorages
// @Summary Get all storages
// @Description Get all storages for a workspace
// @Tags storages
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param workspace_id query string true "Workspace ID"
// @Success 200 {array} Storage
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /storages [get]
func (c *StorageController) GetStorages(ctx *gin.Context) {
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

	storages, err := c.storageService.GetStorages(user, workspaceID)
	if err != nil {
		if err.Error() == "insufficient permissions to view storages in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, storages)
}

// DeleteStorage
// @Summary Delete a storage
// @Description Delete a storage by ID
// @Tags storages
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param id path string true "Storage ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /storages/{id} [delete]
func (c *StorageController) DeleteStorage(ctx *gin.Context) {
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

	if err := c.storageService.DeleteStorage(user, id); err != nil {
		if err.Error() == "insufficient permissions to manage storage in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "storage deleted successfully"})
}

// TestStorageConnection
// @Summary Test storage connection
// @Description Test the connection to the storage
// @Tags storages
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param id path string true "Storage ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /storages/{id}/test [post]
func (c *StorageController) TestStorageConnection(ctx *gin.Context) {
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

	if err := c.storageService.TestStorageConnection(user, id); err != nil {
		if err.Error() == "insufficient permissions to test storage in this workspace" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "storage connection test successful"})
}

// TestStorageConnectionDirect
// @Summary Test storage connection directly
// @Description Test the connection to a storage object provided in the request
// @Tags storages
// @Accept json
// @Produce json
// @Param Authorization header string true "JWT token"
// @Param request body Storage true "Storage data with workspaceId"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 403
// @Router /storages/direct-test [post]
func (c *StorageController) TestStorageConnectionDirect(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Storage
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
			gin.H{"error": "insufficient permissions to test storage in this workspace"},
		)
		return
	}

	if err := c.storageService.TestStorageConnectionDirect(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "storage connection test successful"})
}
