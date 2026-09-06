package verification_agents

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/util/ratelimiter"
)

type failingRateLimitCounter struct {
	err error
}

func (c failingRateLimitCounter) RecordAttemptAndCheckIsAllowed(
	context.Context,
	ratelimiter.Attempt,
) (bool, error) {
	return false, c.err
}

func Test_RequireAgentAuth_WhenRateLimiterFails_RejectsWithoutLoggingAgentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logOutput bytes.Buffer
	service := &AgentService{
		rateLimiter: failingRateLimitCounter{err: errors.New("counter failed")},
		logger:      slog.New(slog.NewTextHandler(&logOutput, nil)),
	}
	router := gin.New()
	router.GET("/agents/:agentId", service.RequireAgentAuth(), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	agentID := uuid.New()
	request := httptest.NewRequest(http.MethodGet, "/agents/"+agentID.String(), nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.NotContains(t, logOutput.String(), agentID.String())
}
