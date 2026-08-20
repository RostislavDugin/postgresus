package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_models "databasus-backend/internal/features/users/models"
	verification_agents "databasus-backend/internal/features/verification/agents"
	"databasus-backend/internal/util/logger"
)

type capturedAccessRecord struct {
	level      slog.Level
	message    string
	requestID  string
	userID     string
	attributes map[string]any
}

type accessRecordCapturer struct {
	records *[]capturedAccessRecord
}

func (h accessRecordCapturer) Enabled(context.Context, slog.Level) bool { return true }

func (h accessRecordCapturer) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h accessRecordCapturer) WithGroup(string) slog.Handler { return h }

func (h accessRecordCapturer) Handle(ctx context.Context, record slog.Record) error {
	captured := capturedAccessRecord{
		level:      record.Level,
		message:    record.Message,
		requestID:  logger.GetRequestID(ctx),
		userID:     logger.GetUserID(ctx),
		attributes: map[string]any{},
	}

	record.Attrs(func(attr slog.Attr) bool {
		captured.attributes[attr.Key] = attr.Value.Any()

		return true
	})

	*h.records = append(*h.records, captured)

	return nil
}

func serveAccessLoggedRoute(t *testing.T, path string, authenticatedUser *users_models.User) capturedAccessRecord {
	t.Helper()

	records := &[]capturedAccessRecord{}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AssignRequestID(), LogAccess(slog.New(accessRecordCapturer{records})))

	if authenticatedUser != nil {
		router.Use(func(ctx *gin.Context) {
			ctx.Set("user", authenticatedUser)
			ctx.Request = ctx.Request.WithContext(
				logger.ContextWithUserID(ctx.Request.Context(), authenticatedUser.ID.String()),
			)
			ctx.Next()
		})
	}

	router.GET("/api/v1/storages", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
	router.GET("/api/v1/databases/:databaseId/crash", func(ctx *gin.Context) {
		ctx.Status(http.StatusInternalServerError)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))

	require.Len(t, *records, 1)
	assert.Equal(t, response.Header().Get(RequestIDHeader), (*records)[0].requestID,
		"the logged request id must be the one the client was given")

	return (*records)[0]
}

func Test_LogAccess_OnSuccessfulRequest_LogsMethodPathAndStatusAtInfo(t *testing.T) {
	record := serveAccessLoggedRoute(t, "/api/v1/storages", nil)

	assert.Equal(t, slog.LevelInfo, record.level)
	assert.Contains(t, record.message, "GET /api/v1/storages -> 200 in ")
	assert.Equal(t, "/api/v1/storages", record.attributes["route"])
}

func Test_LogAccess_WhenHandlerFails_LogsAtErrorLevel(t *testing.T) {
	record := serveAccessLoggedRoute(t, "/api/v1/databases/db-42/crash", nil)

	assert.Equal(t, slog.LevelError, record.level)
	assert.Contains(t, record.message, "-> 500 in ")
	assert.Equal(t, "/api/v1/databases/:databaseId/crash", record.attributes["route"])
}

// The principal reaches the record from the request context, the same way request_id does, so the
// access line is not a second place that has to remember to attach it.
func Test_LogAccess_WhenRequestIsAuthenticated_AttachesUserID(t *testing.T) {
	authenticatedUser := &users_models.User{ID: uuid.New()}

	record := serveAccessLoggedRoute(t, "/api/v1/storages", authenticatedUser)

	assert.Equal(t, authenticatedUser.ID.String(), record.userID)
	assert.NotContains(t, record.attributes, "user_id",
		"the context carries it; attaching it here as well would emit the key twice")
}

// Download and restore tokens travel in the query string, so it must never reach a sink.
func Test_LogAccess_WithQueryString_LeavesItOutOfTheRecord(t *testing.T) {
	record := serveAccessLoggedRoute(t, "/api/v1/storages?token=secret-download-token", nil)

	assert.NotContains(t, record.message, "secret-download-token")
	assert.NotContains(t, record.message, "?")
}

func Test_LogAccess_WhenRequestUsesAgentToken_AttachesAgentID(t *testing.T) {
	authenticatedAgent := &verification_agents.Agent{ID: uuid.New()}

	gin.SetMode(gin.TestMode)

	records := &[]capturedAccessRecord{}
	router := gin.New()
	router.Use(AssignRequestID(), LogAccess(slog.New(accessRecordCapturer{records})))
	router.Use(func(ctx *gin.Context) {
		ctx.Set("verification_agent", authenticatedAgent)
		ctx.Next()
	})
	router.GET("/api/v1/agent/verification/heartbeat", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/v1/agent/verification/heartbeat", nil),
	)

	require.Len(t, *records, 1)
	assert.Equal(t, authenticatedAgent.ID, (*records)[0].attributes["agent_id"])
}

func Test_LogAccess_WhenStaticAssetIsServed_MarksTheRouteAsStatic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	records := &[]capturedAccessRecord{}
	router := gin.New()
	router.Use(AssignRequestID(), LogAccess(slog.New(accessRecordCapturer{records})))
	router.NoRoute(func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/assets/index.js", nil))

	require.Len(t, *records, 1)
	assert.Equal(t, staticRouteName, (*records)[0].attributes["route"])
}

// A handler attaches the cause with ctx.Error instead of logging its own near-duplicate record.
func Test_LogAccess_WhenHandlerAttachesAnError_PutsItOnTheAccessLine(t *testing.T) {
	gin.SetMode(gin.TestMode)

	records := &[]capturedAccessRecord{}
	router := gin.New()
	router.Use(AssignRequestID(), LogAccess(slog.New(accessRecordCapturer{records})))
	router.GET("/api/v1/storages", func(ctx *gin.Context) {
		_ = ctx.Error(errors.New("storage backend unreachable"))
		ctx.Status(http.StatusInternalServerError)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/storages", nil))

	require.Len(t, *records, 1)
	assert.Contains(t, (*records)[0].attributes["error"], "storage backend unreachable")
}
