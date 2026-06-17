package api_keys_middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	api_keys "databasus-backend/internal/features/api_keys"
)

func ApiKeyAuthMiddleware(apiKeyService *api_keys.ApiKeyService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
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

		ctx.Set(api_keys.PrincipalContextKey, principal)
		ctx.Next()
	}
}
