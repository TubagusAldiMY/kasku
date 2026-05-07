package http

import (
	"net/http"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// NewRouter membuat dan mengkonfigurasi Gin router dengan semua middleware dan routes.
func NewRouter(authHandler *handler.AuthHandler, isDev bool, logger zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Recovery middleware — mencegah crash dari panic
	r.Use(gin.Recovery())

	// Correlation ID — inject ke setiap request
	r.Use(middleware.CorrelationID())

	// Security headers — OWASP best practices
	r.Use(securityHeaders())

	// Request logger — structured JSON logging
	r.Use(requestLogger(logger))

	// Health check (public, tanpa auth)
	r.GET("/health", authHandler.Health)

	// Auth endpoints
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/verify-email", authHandler.VerifyEmail) // query param: ?token=
		auth.POST("/resend-verification", authHandler.ResendVerification)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
	}

	return r
}

// securityHeaders meng-inject security headers ke setiap response.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// requestLogger mencatat setiap request HTTP dalam format JSON terstruktur.
func requestLogger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		event := logger.Info()
		if statusCode >= http.StatusInternalServerError {
			event = logger.Error()
		} else if statusCode >= http.StatusBadRequest {
			event = logger.Warn()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", statusCode).
			Dur("duration_ms", duration).
			Str("correlation_id", middleware.GetCorrelationID(c)).
			Str("client_ip", c.ClientIP()).
			Msg("http request")
	}
}
