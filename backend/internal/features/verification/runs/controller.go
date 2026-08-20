package verification_runs

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_middleware "databasus-backend/internal/features/users/middleware"
)

type VerificationController struct {
	verificationService *VerificationService
}

func (c *VerificationController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/verifications/enqueue", c.EnqueueManual)
	router.POST("/verifications/:id/cancel", c.Cancel)
	router.GET("/verifications/:id", c.GetByID)
	router.GET("/verifications/by-database/:databaseId", c.ListByDatabase)
}

// EnqueueManual
// @Summary Enqueue manual backup verification
// @Description Trigger a one-off verification of the chosen backup. Returns 400 if a manual verification is already pending or running for the same database — cancel it first.
// @Tags verifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body EnqueueManualRequest true "Manual verification request"
// @Success 200 {object} RestoreVerification
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /verifications/enqueue [post]
func (c *VerificationController) EnqueueManual(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req EnqueueManualRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	verification, err := c.verificationService.EnqueueManualVerification(ctx.Request.Context(), user, req.BackupID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, verification)
}

// Cancel
// @Summary Cancel a pending or running verification
// @Description Marks a PENDING or RUNNING verification as CANCELED. For RUNNING rows the agent stops on its next heartbeat. Returns 400 if the verification is already terminal.
// @Tags verifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Verification ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /verifications/{id}/cancel [post]
func (c *VerificationController) Cancel(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	verificationID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	if err := c.verificationService.CancelVerification(ctx.Request.Context(), user, verificationID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetByID
// @Summary Get verification by id
// @Description Returns one verification row with its per-table stats preloaded.
// @Tags verifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Verification ID"
// @Success 200 {object} RestoreVerification
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /verifications/{id} [get]
func (c *VerificationController) GetByID(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	verificationID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	verification, err := c.verificationService.GetVerificationByID(ctx.Request.Context(), user, verificationID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, verification)
}

// ListByDatabase
// @Summary List verifications by database
// @Description Returns verifications for the given database, newest first. Per-table stats are NOT included — use GetByID for that.
// @Tags verifications
// @Produce json
// @Security BearerAuth
// @Param databaseId path string true "Database ID"
// @Param limit query int false "Limit number of results" default(10)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} GetVerificationsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /verifications/by-database/{databaseId} [get]
func (c *VerificationController) ListByDatabase(ctx *gin.Context) {
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

	var req GetVerificationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.verificationService.GetVerificationsByDatabaseID(
		ctx.Request.Context(),
		user,
		databaseID,
		req.Limit,
		req.Offset,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
