package api_keys_middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	api_keys "databasus-backend/internal/features/api_keys"
	cache_utils "databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/logger"
)

const (
	rateLimitWindow = time.Minute

	// maxRequestsPerIP backstops token probing before auth; looser than the per-key limit.
	maxRequestsPerIP = 240

	// maxRequestsPerKey bounds one key's call rate (each trigger starts a real backup);
	// tighter than the per-IP limit so it trips first for a valid key.
	maxRequestsPerKey = 120

	rateLimitEndpointByIP  = "public_api_key_by_ip"
	rateLimitEndpointByKey = "public_api_key_by_key"
)

func GetApiKeyAuthMiddleware() gin.HandlerFunc {
	return ApiKeyAuthMiddleware(
		api_keys.GetApiKeyService(),
		cache_utils.NewRateLimiter(cache_utils.GetValkeyClient()),
	)
}

func ApiKeyAuthMiddleware(
	apiKeyService *api_keys.ApiKeyService,
	rateLimiter *cache_utils.RateLimiter,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Key on RemoteIP, not ClientIP, so a spoofed X-Forwarded-For cannot reset the
		// pre-auth bucket. Behind a trusted proxy, set SetTrustedProxies for the real edge IP.
		if !isWithinRateLimit(rateLimiter, ctx.RemoteIP(), rateLimitEndpointByIP, maxRequestsPerIP) {
			rejectRateLimited(ctx)

			return
		}

		token := ctx.GetHeader("Authorization")
		if token == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			ctx.Abort()

			return
		}

		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		principal, err := apiKeyService.AuthenticateToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			ctx.Abort()

			return
		}

		if !isWithinRateLimit(rateLimiter, principal.ApiKeyID.String(), rateLimitEndpointByKey, maxRequestsPerKey) {
			rejectRateLimited(ctx)

			return
		}

		ctx.Set(api_keys.PrincipalContextKey, principal)
		ctx.Next()
	}
}

// isWithinRateLimit fails open when the limiter backend is down, logging so the degradation is visible.
func isWithinRateLimit(
	rateLimiter *cache_utils.RateLimiter,
	identifier string,
	endpoint string,
	maxRequests int,
) bool {
	allowed, err := rateLimiter.CheckLimit(identifier, endpoint, maxRequests, rateLimitWindow)
	if err != nil {
		logger.GetLogger().Warn(
			"public api key rate limiter unavailable, allowing request",
			"endpoint", endpoint,
			"error", err,
		)
	}

	return allowed
}

func rejectRateLimited(ctx *gin.Context) {
	ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded. Please try again later."})
	ctx.Abort()
}
