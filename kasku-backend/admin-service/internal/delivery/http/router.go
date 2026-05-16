package http

import (
	"net/http"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func NewRouter(healthHandler *handler.HealthHandler, isDev bool, logger zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(securityHeaders())
	r.Use(requestLogger(logger))

	r.GET("/health", healthHandler.Health)
	r.GET("/metrics", metrics("admin-service"))

	admin := r.Group("/admin")
	{
		admin.GET("/health", healthHandler.Health)
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

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

func requestLogger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		event := logger.Info()
		if c.Writer.Status() >= http.StatusInternalServerError {
			event = logger.Error()
		} else if c.Writer.Status() >= http.StatusBadRequest {
			event = logger.Warn()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("duration", time.Since(start)).
			Msg("http request")
	}
}
