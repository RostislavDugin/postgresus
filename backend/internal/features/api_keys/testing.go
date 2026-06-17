package api_keys

import (
	"github.com/gin-gonic/gin"

	audit_logs "databasus-backend/internal/features/audit_logs"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
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
