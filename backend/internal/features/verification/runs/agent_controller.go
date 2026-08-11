package verification_runs

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	verification_agents "databasus-backend/internal/features/verification/agents"
)

type VerificationAgentController struct {
	verificationService *VerificationService
	agentService        *verification_agents.AgentService
	logger              *slog.Logger
}

func (c *VerificationAgentController) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/agent/verifications/:agentId")
	g.Use(c.agentService.RequireAgentAuth())

	g.POST("/claim", c.Claim)
	g.GET("/:id/backup-stream", c.BackupStream)
	g.POST("/:id/report", c.Report)
}

// Claim
// @Summary Claim next verification job
// @Description Called by a verification agent. Returns the oldest PENDING verification that fits the agent's disk budget, or 204 if nothing fits.
// @Tags verification-agents
// @Accept json
// @Produce json
// @Param agentId path string true "Agent UUID"
// @Param Authorization header string true "Bearer <agent-token>"
// @Param request body ClaimRequest true "Agent capacity"
// @Success 200 {object} JobAssignment
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /agent/verifications/{agentId}/claim [post]
func (c *VerificationAgentController) Claim(ctx *gin.Context) {
	agent, ok := verification_agents.GetAgentFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "agent missing from context"})
		return
	}

	var req ClaimRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	assignment, err := c.verificationService.ClaimVerification(agent, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if assignment == nil {
		ctx.Status(http.StatusNoContent)
		return
	}

	ctx.JSON(http.StatusOK, assignment)
}

// BackupStream
// @Summary Stream the backup file for a verification
// @Description Streams the decrypted backup file to the agent that owns this RUNNING verification. Same bytes the user-facing /backups/{id}/download endpoint returns.
// @Tags verification-agents
// @Produce application/octet-stream
// @Param agentId path string true "Agent UUID"
// @Param id path string true "Verification UUID"
// @Param Authorization header string true "Bearer <agent-token>"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 410 {object} map[string]string
// @Router /agent/verifications/{agentId}/{id}/backup-stream [get]
func (c *VerificationAgentController) BackupStream(ctx *gin.Context) {
	agent, ok := verification_agents.GetAgentFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "agent missing from context"})
		return
	}

	verificationID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	reader, err := c.verificationService.GetBackupFile(ctx.Request.Context(), agent, verificationID)
	if err != nil {
		ctx.JSON(http.StatusGone, gin.H{"error": err.Error(), "reason": "gone"})
		return
	}

	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			c.logger.Error(
				"failed to close backup reader",
				"error", closeErr,
				"verification_id", verificationID,
			)
		}
	}()

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="verification-%s.dump"`, verificationID),
	)

	if _, err := io.Copy(ctx.Writer, reader); err != nil {
		c.logger.Error(
			"failed to stream backup to agent",
			"error", err,
			"verification_id", verificationID,
		)
	}
}

// Report
// @Summary Submit verification result
// @Description Agent reports COMPLETED (with stats) or FAILED. CAS-guarded: stale or foreign reports get 410.
// @Tags verification-agents
// @Accept json
// @Produce json
// @Param agentId path string true "Agent UUID"
// @Param id path string true "Verification UUID"
// @Param Authorization header string true "Bearer <agent-token>"
// @Param request body ReportRequest true "Report payload"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 410 {object} map[string]string
// @Router /agent/verifications/{agentId}/{id}/report [post]
func (c *VerificationAgentController) Report(ctx *gin.Context) {
	agent, ok := verification_agents.GetAgentFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "agent missing from context"})
		return
	}

	verificationID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	var req ReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.verificationService.SubmitReport(agent, verificationID, &req); err != nil {
		if err.Error() == "verification gone" {
			ctx.JSON(http.StatusGone, gin.H{"reason": "gone"})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}
