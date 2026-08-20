package disk

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type DiskController struct {
	diskService *DiskService
}

func (c *DiskController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/disk/usage", c.GetDiskUsage)
}

// GetDiskUsage
// @Summary Get disk usage information
// @Description Returns information about disk space usage
// @Tags disk
// @Produce json
// @Success 200 {object} DiskUsage
// @Failure 500
// @Router /disk/usage [get]
func (c *DiskController) GetDiskUsage(ctx *gin.Context) {
	diskUsage, err := c.diskService.GetDiskUsage(ctx.Request.Context())
	if err != nil {
		_ = ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, diskUsage)
}
