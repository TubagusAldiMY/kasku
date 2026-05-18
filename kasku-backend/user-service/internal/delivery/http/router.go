package http

import (
	"github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/TubagusAldiMY/kasku/user-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/user-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func NewRouter(userHandler *handler.UserHandler, isDev bool, metricsReg *metrics.Registry, log zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())
	r.Use(metricsReg.HTTPMetrics())

	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	r.GET("/health", userHandler.Health)
	r.GET("/metrics", gin.WrapH(metricsReg.Handler()))

	v1 := r.Group("/v1")
	{
		users := v1.Group("/users")
		{
			users.GET("/profile", userHandler.GetProfile)
			users.PUT("/profile", userHandler.UpdateProfile)
		}
	}

	return r
}
