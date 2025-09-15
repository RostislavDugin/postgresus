package sqlquery

import (
	"net/http"

	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/users"

	"github.com/gin-gonic/gin"
)

type PostgresSqlQueryController struct {
	svc     *Service
	userSvc *users.UserService
	dbSvc   *databases.DatabaseService
}

func NewPostgresSqlQueryController(svc *Service, userSvc *users.UserService, dbSvc *databases.DatabaseService) *PostgresSqlQueryController {
	return &PostgresSqlQueryController{svc: svc, userSvc: userSvc, dbSvc: dbSvc}
}

func (c *PostgresSqlQueryController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/sqlquery/execute", c.Execute)
}

func (c *PostgresSqlQueryController) Execute(ctx *gin.Context) {
	var req ExecuteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	auth := ctx.GetHeader("Authorization")
	if auth == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
		return
	}
	user, err := c.userSvc.GetUserFromToken(auth)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	dbc, err := c.dbSvc.GetDatabase(user, req.DatabaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := c.svc.Execute(dbc, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}