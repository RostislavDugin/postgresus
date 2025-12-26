package metrics

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsController struct{}

func NewMetricsController() *MetricsController {
	return &MetricsController{}
}

func (c *MetricsController) RegisterRoutes(router *gin.RouterGroup) {
	// Expose Prometheus metrics endpoint
	// This should be public (no auth) as Prometheus scrapes it
	router.GET("/metrics", c.GetMetrics)
}

// GetMetrics exposes Prometheus metrics
// @Summary Get Prometheus metrics
// @Description Exposes Prometheus metrics endpoint for monitoring
// @Tags metrics
// @Produce text/plain
// @Success 200 {string} string "Prometheus metrics"
// @Router /metrics [get]
func (c *MetricsController) GetMetrics(ctx *gin.Context) {
	// Use promhttp handler to serve metrics
	handler := promhttp.Handler()
	handler.ServeHTTP(ctx.Writer, ctx.Request)
}

// GetMetricsHandler returns the HTTP handler for metrics
// This can be used directly with standard http handlers
func GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}

