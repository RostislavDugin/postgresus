package clusters

import (
	"net/http"
	users_middleware "postgresus-backend/internal/features/users/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ClusterController struct {
	service *ClusterService
}

func (c *ClusterController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/clusters", c.GetClusters)
	router.POST("/clusters", c.CreateCluster)
	router.POST("/clusters/:id/run-backup", c.RunClusterBackup)
	router.PUT("/clusters/:id", c.UpdateCluster)
	router.GET("/clusters/:id/databases", c.ListClusterDatabases)
	router.GET("/clusters/:id/propagation/preview", c.PreviewPropagation)
	router.POST("/clusters/:id/propagation/apply", c.ApplyPropagation)
}

// CreateCluster
// @Summary Create a new cluster
// @Tags clusters
// @Accept json
// @Produce json
// @Param request body Cluster true "Cluster creation data with workspaceId"
// @Success 201 {object} Cluster
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /clusters [post]
func (c *ClusterController) CreateCluster(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request Cluster
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.WorkspaceID == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "workspaceId is required"})
		return
	}

	cluster, err := c.service.CreateCluster(user, *request.WorkspaceID, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cluster.HideSensitiveData()
	ctx.JSON(http.StatusCreated, cluster)
}

// GetClusters
// @Summary Get clusters by workspace
// @Tags clusters
// @Produce json
// @Param workspace_id query string true "Workspace ID"
// @Success 200 {array} Cluster
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /clusters [get]
func (c *ClusterController) GetClusters(ctx *gin.Context) {
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

	clusters, err := c.service.GetClusters(user, workspaceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, clusters)
}

// RunClusterBackup
// @Summary Run backup across a cluster (discovering new DBs)
// @Tags clusters
// @Param id path string true "Cluster ID"
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /clusters/{id}/run-backup [post]
func (c *ClusterController) RunClusterBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid cluster ID"})
		return
	}

	if err := c.service.RunBackup(user, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "cluster backup started"})
}

// UpdateCluster
// @Summary Update cluster
// @Tags clusters
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body Cluster true "Cluster update data"
// @Success 200 {object} Cluster
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /clusters/{id} [put]
func (c *ClusterController) UpdateCluster(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid cluster ID"})
		return
	}

	var request Cluster
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := c.service.UpdateCluster(user, id, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updated)
}

// ListClusterDatabases
// @Summary List accessible databases of a cluster
// @Tags clusters
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} map[string][]string
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /clusters/{id}/databases [get]
func (c *ClusterController) ListClusterDatabases(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid cluster ID"})
		return
	}

	dbs, err := c.service.ListClusterDatabases(user, id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"databases": dbs})
}

// PreviewPropagation
// @Summary Preview force-apply of cluster defaults to existing DBs
// @Tags clusters
// @Produce json
// @Param id path string true "Cluster ID"
// @Param applyStorage query bool false "Apply storage"
// @Param applySchedule query bool false "Apply schedule"
// @Param respectExclusions query bool false "Respect cluster exclusions"
// @Success 200 {array} SwaggerPropagationChange
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /clusters/{id}/propagation/preview [get]
func (c *ClusterController) PreviewPropagation(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid cluster ID"})
		return
	}

	q := ctx.Request.URL.Query()
	opts := PropagationOptions{
		ApplyStorage:       q.Get("applyStorage") == "true" || q.Get("apply_storage") == "true" || q.Get("apply_storage") == "1",
		ApplySchedule:      q.Get("applySchedule") == "true" || q.Get("apply_schedule") == "true" || q.Get("apply_schedule") == "1",
		ApplyEnableBackups: q.Get("applyEnableBackups") == "true" || q.Get("apply_enable_backups") == "true" || q.Get("apply_enable_backups") == "1",
		RespectExclusions:  q.Get("respectExclusions") == "true" || q.Get("respect_exclusions") == "true" || q.Get("respect_exclusions") == "1",
	}

	res, err := c.service.PreviewPropagation(user, id, opts)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, res)
}

type applyPropagationRequest struct {
	ApplyStorage       bool `json:"applyStorage"`
	ApplySchedule      bool `json:"applySchedule"`
	ApplyEnableBackups bool `json:"applyEnableBackups"`
	RespectExclusions  bool `json:"respectExclusions"`
}

// SwaggerPropagationChange is a copy of PropagationChange for Swagger generation
// It avoids cross-file/type resolution issues during swag parsing.
type SwaggerPropagationChange struct {
	DatabaseID     uuid.UUID `json:"databaseId"`
	Name           string    `json:"name"`
	ChangeStorage  bool      `json:"changeStorage"`
	ChangeSchedule bool      `json:"changeSchedule"`
	ChangeEnabled  bool      `json:"changeEnabled"`
}

// ApplyPropagation
// @Summary Force-apply cluster defaults to existing DBs
// @Tags clusters
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body applyPropagationRequest true "Propagation options"
// @Success 200 {array} SwaggerPropagationChange
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /clusters/{id}/propagation/apply [post]
func (c *ClusterController) ApplyPropagation(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid cluster ID"})
		return
	}

	var req applyPropagationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	opts := PropagationOptions{
		ApplyStorage:       req.ApplyStorage,
		ApplySchedule:      req.ApplySchedule,
		ApplyEnableBackups: req.ApplyEnableBackups,
		RespectExclusions:  req.RespectExclusions,
	}

	res, err := c.service.ApplyPropagation(user, id, opts)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, res)
}
