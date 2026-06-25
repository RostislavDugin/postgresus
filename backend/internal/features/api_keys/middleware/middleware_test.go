package api_keys_middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api_keys "databasus-backend/internal/features/api_keys"
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_testing "databasus-backend/internal/features/users/testing"
	cache_utils "databasus-backend/internal/util/cache"
)

// newPublicApiKeyRouter wires the real middleware (not a mirror) in front of a trivial handler.
func newPublicApiKeyRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	rateLimiter := cache_utils.NewRateLimiter(cache_utils.GetValkeyClient())

	router := gin.New()
	public := router.Group("")
	public.Use(ApiKeyAuthMiddleware(api_keys.GetApiKeyService(), rateLimiter))
	public.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	return router
}

// randomTestIP gives each run a fresh source IP so per-IP buckets don't accumulate across runs.
func randomTestIP() string {
	id := uuid.New()

	return fmt.Sprintf("10.%d.%d.%d", id[0], id[1], id[2])
}

func createAdminApiKeyToken(t *testing.T) string {
	t.Helper()

	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	adminUser := &users_models.User{ID: admin.UserID, Role: users_enums.UserRoleAdmin}

	created, err := api_keys.GetApiKeyService().CreateApiKey(adminUser, &api_keys.CreateApiKeyRequestDTO{
		Name: "ratelimit-" + uuid.New().String(),
		Role: users_enums.UserRoleAdmin,
	})
	require.NoError(t, err)

	return created.Token
}

func sendPing(t *testing.T, router *gin.Engine, token string, clientIP string) int {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.RemoteAddr = clientIP + ":54321"
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	router.ServeHTTP(recorder, request)

	return recorder.Code
}

// A fresh key has an empty per-key bucket, so the per-key limit (tighter than per-IP) trips deterministically.
func Test_ApiKeyAuthMiddleware_WhenPerKeyLimitExceeded_Returns429(t *testing.T) {
	router := newPublicApiKeyRouter()
	token := createAdminApiKeyToken(t)
	clientIP := randomTestIP()

	for range maxRequestsPerKey {
		require.Equal(t, http.StatusOK, sendPing(t, router, token, clientIP))
	}

	assert.Equal(t, http.StatusTooManyRequests, sendPing(t, router, token, clientIP))
}

// The per-IP limit must trip before authentication, bounding unauthenticated token probing.
func Test_ApiKeyAuthMiddleware_WhenPerIPLimitExceeded_Returns429BeforeAuth(t *testing.T) {
	router := newPublicApiKeyRouter()
	clientIP := randomTestIP()

	for range maxRequestsPerIP {
		require.Equal(t, http.StatusUnauthorized, sendPing(t, router, "", clientIP))
	}

	assert.Equal(t, http.StatusTooManyRequests, sendPing(t, router, "", clientIP))
}

func Test_ApiKeyAuthMiddleware_WhenTokenMissing_Returns401(t *testing.T) {
	router := newPublicApiKeyRouter()

	assert.Equal(t, http.StatusUnauthorized, sendPing(t, router, "", randomTestIP()))
}
