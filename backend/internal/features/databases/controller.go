package databases

import (
	"net/http"
	users_middleware "postgresus-backend/internal/features/users/middleware"
	users_services "postgresus-backend/internal/features/users/services"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DatabaseController struct {
	databaseService  *DatabaseService
	userService      *users_services.UserService
	workspaceService *workspaces_services.WorkspaceService
}

func (c *DatabaseController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/databases/create", c.CreateDatabase)
	router.POST("/databases/update", c.UpdateDatabase)
	router.DELETE("/databases/:id", c.DeleteDatabase)
	router.GET("/databases/:id", c.GetDatabase)
	router.GET("/databases", c.GetDatabases)
	router.POST("/databases/:id/test-connection", c.TestDatabaseConnection)
	router.POST("/databases/test-connection-direct", c.TestDatabaseConnectionDirect)
	router.POST("/databases/:id/copy", c.CopyDatabase)
	router.GET("/databases/notifier/:id/is-using", c.IsNotifierUsing)
	router.POST("/databases/is-readonly", c.IsUserReadOnly)
	router.POST("/databases/create-readonly-user", c.CreateReadOnlyUser)
}

// CreateDatabase
// @Summary Create a new database
// @Description Create a new database configuration in a workspace
// @Tags databases
// @Accept json
// @Produce json
// @Param request body Database true "Database creation data with workspaceId"
// @Success 201 {object} Database
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /databases/create [post]
func (c *DatabaseController) CreateDatabase(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	database, err := c.databaseService.CreateDatabase(user, *request.WorkspaceID, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, database)
}

// UpdateDatabase
// @Summary Update a database
// @Description Update an existing database configuration
// @Tags databases
// @Accept json
// @Produce json
// @Param request body Database true "Database update data"
// @Success 200 {object} Database
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /databases/update [post]
func (c *DatabaseController) UpdateDatabase(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.databaseService.UpdateDatabase(user, &request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, request)
}

// DeleteDatabase
// @Summary Delete a database
// @Description Delete a database configuration
// @Tags databases
// @Param id path string true "Database ID"
// @Success 204
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /databases/{id} [delete]
func (c *DatabaseController) DeleteDatabase(ctx *gin.Context) {
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

	if err := c.databaseService.DeleteDatabase(user, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetDatabase
// @Summary Get a database
// @Description Get a database configuration by ID
// @Tags databases
// @Produce json
// @Param id path string true "Database ID"
// @Success 200 {object} Database
// @Failure 400
// @Failure 401
// @Router /databases/{id} [get]
func (c *DatabaseController) GetDatabase(ctx *gin.Context) {
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

	database, err := c.databaseService.GetDatabase(user, id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, database)
}

// GetDatabases
// @Summary Get databases by workspace
// @Description Get all databases for a specific workspace
// @Tags databases
// @Produce json
// @Param workspace_id query string true "Workspace ID"
// @Success 200 {array} Database
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /databases [get]
func (c *DatabaseController) GetDatabases(ctx *gin.Context) {
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

	databases, err := c.databaseService.GetDatabasesByWorkspace(user, workspaceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, databases)
}

// TestDatabaseConnection
// @Summary Test database connection
// @Description Test connection to an existing database configuration
// @Tags databases
// @Param id path string true "Database ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /databases/{id}/test-connection [post]
func (c *DatabaseController) TestDatabaseConnection(ctx *gin.Context) {
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

	if err := c.databaseService.TestDatabaseConnection(user, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "connection successful"})
}

// TestDatabaseConnectionDirect
// @Summary Test database connection directly
// @Description Test connection to a database configuration without saving it
// @Tags databases
// @Accept json
// @Param request body Database true "Database configuration to test"
// @Success 200
// @Failure 400
// @Failure 401
// @Router /databases/test-connection-direct [post]
func (c *DatabaseController) TestDatabaseConnectionDirect(ctx *gin.Context) {
	_, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.databaseService.TestDatabaseConnectionDirect(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "connection successful"})
}

// IsNotifierUsing
// @Summary Check if notifier is being used
// @Description Check if a notifier is currently being used by any database
// @Tags databases
// @Produce json
// @Param id path string true "Notifier ID"
// @Success 200 {object} map[string]bool
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /databases/notifier/{id}/is-using [get]
func (c *DatabaseController) IsNotifierUsing(ctx *gin.Context) {
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

	isUsing, err := c.databaseService.IsNotifierUsing(user, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"isUsing": isUsing})
}

// CopyDatabase
// @Summary Copy a database
// @Description Copy an existing database configuration
// @Tags databases
// @Produce json
// @Param id path string true "Database ID"
// @Success 201 {object} Database
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /databases/{id}/copy [post]
func (c *DatabaseController) CopyDatabase(ctx *gin.Context) {
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

	copiedDatabase, err := c.databaseService.CopyDatabase(user, id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, copiedDatabase)
}

// IsUserReadOnly
// @Summary Check if database user is read-only
// @Description Check if current database credentials have only read (SELECT) privileges
// @Tags databases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body Database true "Database configuration to check"
// @Success 200 {object} IsReadOnlyResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /databases/is-readonly [post]
func (c *DatabaseController) IsUserReadOnly(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isReadOnly, err := c.databaseService.IsUserReadOnly(user, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, IsReadOnlyResponse{IsReadOnly: isReadOnly})
}

// CreateReadOnlyUser
// @Summary Create read-only database user
// @Description Create a new PostgreSQL user with read-only privileges for backup operations
// @Tags databases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body Database true "Database configuration to create user for"
// @Success 200 {object} CreateReadOnlyUserResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /databases/create-readonly-user [post]
func (c *DatabaseController) CreateReadOnlyUser(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Database
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username, password, err := c.databaseService.CreateReadOnlyUser(user, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, CreateReadOnlyUserResponse{
		Username: username,
		Password: password,
	})
}
