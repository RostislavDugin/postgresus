package verification_agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AgentFacingController struct {
	agentService *AgentService
}

func (c *AgentFacingController) RegisterRoutes(router *gin.RouterGroup) {
	g := router.Group("/agent/verification/:agentId")
	g.Use(c.agentService.RequireAgentAuth())

	g.POST("/heartbeat", c.Heartbeat)
}

// Heartbeat
// @Summary Verification agent heartbeat
// @Description Called periodically by a verification agent to report capacity and refresh last-seen.
// @Description Authentication uses the agent's raw token (NOT a user JWT) — the same token returned at create/rotate.
// @Tags verification-agents
// @Accept json
// @Produce json
// @Param agentId path string true "Agent UUID"
// @Param Authorization header string true "Bearer <agent-token>"
// @Param request body HeartbeatRequest true "Capacity report"
// @Success 200 {object} HeartbeatResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 429 {object} map[string]string
// @Router /agent/verification/{agentId}/heartbeat [post]
func (c *AgentFacingController) Heartbeat(ctx *gin.Context) {
	agent, ok := GetAgentFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "agent missing from context"})
		return
	}

	var request HeartbeatRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	seenAt, abortVerificationIDs, err := c.agentService.Heartbeat(ctx.Request.Context(), agent, &request)
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, HeartbeatResponse{
		LastSeenAt:           seenAt,
		AbortVerificationIDs: abortVerificationIDs,
	})
}
