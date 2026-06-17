package api_keys

import (
	"net/http"

	"github.com/gin-gonic/gin"

	audit_logs "databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/databases"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
)

func CreateApiKeyTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("")
	protected.Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	GetApiKeyController().RegisterRoutes(protected)

	audit_logs.SetupDependencies()

	return router
}

func CreateApiKeyPublicTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	protected := v1.Group("")
	protected.Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetApiKeyController().RegisterRoutes(protected)
	workspaces_controllers.GetWorkspaceController().RegisterRoutes(protected)
	workspaces_controllers.GetMembershipController().RegisterRoutes(protected)
	databases.GetDatabaseController().RegisterRoutes(protected)

	apiKeySvc := GetApiKeyService()
	public := v1.Group("")
	public.Use(func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if token == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			ctx.Abort()

			return
		}

		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		principal, err := apiKeySvc.AuthenticateToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			ctx.Abort()

			return
		}

		ctx.Set(PrincipalContextKey, principal)
		ctx.Next()
	})
	GetApiKeyController().RegisterPublicRoutes(public)

	audit_logs.SetupDependencies()

	return router
}
