package http

import (
	"github.com/gin-gonic/gin"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/delivery/http/middleware"
)

// NewRouter membuat dan mengkonfigurasi Gin router untuk investment-service.
func NewRouter(h *handler.InvestmentHandler, isDev bool) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())

	// Health check (public)
	r.GET("/health", h.Health)

	// Investment endpoints
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
