package users_middleware

import (
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_services "databasus-backend/internal/features/users/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT token and adds user to context
func AuthMiddleware(userService *users_services.UserService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if token == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			ctx.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		user, err := userService.GetUserFromToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			ctx.Abort()
			return
		}

		ctx.Set("user", user)
		ctx.Next()
	}
}

func RequireRole(requiredRole users_enums.UserRole) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userInterface, exists := ctx.Get("user")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			ctx.Abort()
			return
		}

		user, ok := userInterface.(*users_models.User)
		if !ok {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			ctx.Abort()
			return
		}

		if user.Role != requiredRole {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// GetUserFromContext helper function to extract user from gin context
func GetUserFromContext(ctx *gin.Context) (*users_models.User, bool) {
	userInterface, exists := ctx.Get("user")
	if !exists {
		return nil, false
	}

	user, ok := userInterface.(*users_models.User)

	return user, ok
}
