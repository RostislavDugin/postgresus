package workspaces_testing

import "github.com/gin-gonic/gin"

type ControllerInterface interface {
	RegisterRoutes(router *gin.RouterGroup)
}
