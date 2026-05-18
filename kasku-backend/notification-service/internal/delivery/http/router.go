package http

import (
	"github.com/TubagusAldiMY/kasku/notification-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func NewRouter(healthHandler *handler.HealthHandler, preferenceHandler *handler.PreferenceHandler, isDev bool, metricsReg *metrics.Registry, log zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(metricsReg.HTTPMetrics())
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	})

	r.GET("/health", healthHandler.Health)
	r.GET("/metrics", gin.WrapH(metricsReg.Handler()))

	v1 := r.Group("/v1/notifications")
	{
		v1.GET("/preferences", preferenceHandler.Get)
		v1.PUT("/preferences", preferenceHandler.Update)
	}

	return r
}
