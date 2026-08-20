package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/logger"
)

func buildRouterWithRequestID() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AssignRequestID())
	router.GET("/api/v1/system/version", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, logger.GetRequestID(ctx.Request.Context()))
	})

	return router
}

func serveRequestIDRoute(inboundRequestID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/version", nil)
	if inboundRequestID != "" {
		request.Header.Set(RequestIDHeader, inboundRequestID)
	}

	response := httptest.NewRecorder()
	buildRouterWithRequestID().ServeHTTP(response, request)

	return response
}

func Test_AssignRequestID_OnEveryRequest_EchoesTheIDAndPutsItInTheContext(t *testing.T) {
	response := serveRequestIDRoute("")

	requestID := response.Header().Get(RequestIDHeader)
	_, err := uuid.Parse(requestID)
	require.NoError(t, err, "the echoed request id must be a UUID")

	assert.Equal(t, requestID, response.Body.String(), "handlers must see the same id as the client")
}

func Test_AssignRequestID_AcrossRequests_GeneratesADistinctID(t *testing.T) {
	firstRequestID := serveRequestIDRoute("").Header().Get(RequestIDHeader)
	secondRequestID := serveRequestIDRoute("").Header().Get(RequestIDHeader)

	assert.NotEqual(t, firstRequestID, secondRequestID)
}

func Test_AssignRequestID_WhenClientSendsOne_IgnoresIt(t *testing.T) {
	clientRequestID := "11111111-1111-1111-1111-111111111111"

	response := serveRequestIDRoute(clientRequestID)

	assert.NotEqual(t, clientRequestID, response.Header().Get(RequestIDHeader))
	assert.NotEqual(t, clientRequestID, response.Body.String())
}
