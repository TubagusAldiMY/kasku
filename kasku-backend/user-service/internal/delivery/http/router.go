package http

import (
	"net/http"

	"github.com/TubagusAldiMY/kasku/user-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/user-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func NewRouter(userHandler *handler.UserHandler, isDev bool, log zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())

	// Security headers
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	r.GET("/health", userHandler.Health)
	r.GET("/metrics", metrics("user-service"))

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

func metrics(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(
			"# HELP kasku_service_info KasKu service metadata\n"+
				"# TYPE kasku_service_info gauge\n"+
				"kasku_service_info{service=\""+service+"\"} 1\n",
		))
	}
}
