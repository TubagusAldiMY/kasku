package http

import (
	"github.com/TubagusAldiMY/kasku/notification-service/internal/delivery/http/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// NewRouter membuat Gin engine dengan security headers dan health endpoint.
func NewRouter(healthHandler *handler.HealthHandler, isDev bool, log zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	})

	r.GET("/health", healthHandler.Health)
	return r
}
