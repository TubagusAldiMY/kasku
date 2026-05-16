package http

import (
	"net/http"

	"github.com/TubagusAldiMY/kasku/notification-service/internal/delivery/http/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// NewRouter membuat Gin engine dengan security headers dan health endpoint.
func NewRouter(healthHandler *handler.HealthHandler, preferenceHandler *handler.PreferenceHandler, isDev bool, log zerolog.Logger) *gin.Engine {
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
	r.GET("/metrics", metrics("notification-service"))

	v1 := r.Group("/v1/notifications")
	{
		v1.GET("/preferences", preferenceHandler.Get)
		v1.PUT("/preferences", preferenceHandler.Update)
	}

	return r
}

func metrics(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(
			"# HELP kasku_service_info KasKu service metadata\n"+
				"# TYPE kasku_service_info gauge\n"+
				"kasku_service_info{service=\""+service+"\"} 1\n",
		))
	}
}
