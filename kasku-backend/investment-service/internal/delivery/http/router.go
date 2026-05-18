package http

import (
	"github.com/TubagusAldiMY/kasku/investment-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/gin-gonic/gin"
)

func NewRouter(h *handler.InvestmentHandler, isDev bool, metricsReg *metrics.Registry) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())
	r.Use(metricsReg.HTTPMetrics())

	r.GET("/health", h.Health)
	r.GET("/metrics", gin.WrapH(metricsReg.Handler()))

	v1 := r.Group("/v1/investments")
	{
		v1.GET("", h.ListAssets)
		v1.POST("", h.CreateAsset)
		v1.GET("/:id", h.GetAsset)
		v1.PUT("/:id", h.UpdateAsset)
		v1.DELETE("/:id", h.DeleteAsset)
		v1.POST("/:id/units", h.RecordUnitChange)
		v1.GET("/:id/history", h.GetUnitHistory)
	}

	return r
}
