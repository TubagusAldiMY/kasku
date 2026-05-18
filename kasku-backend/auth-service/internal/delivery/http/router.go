package http

import (
	"net/http"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/configs"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/ratelimit"
	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// NewRouter membuat dan mengkonfigurasi Gin router dengan semua middleware dan routes.
func NewRouter(
	authHandler *handler.AuthHandler,
	cfg *configs.Config,
	limiter ratelimit.Limiter,
	metricsReg *obsmetrics.Registry,
	logger zerolog.Logger,
) *gin.Engine {
	isDev := cfg.IsDevelopment()
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Recovery middleware — mencegah crash dari panic
	r.Use(gin.Recovery())

	// Correlation ID — inject ke setiap request
	r.Use(middleware.CorrelationID())

	// Prometheus HTTP metrics — request count + duration per route
	r.Use(metricsReg.HTTPMetrics())

	// Security headers — OWASP best practices
	r.Use(securityHeaders())

	// Request logger — structured JSON logging
	r.Use(requestLogger(logger))

	// Health check (public, tanpa auth)
	r.GET("/health", authHandler.Health)
	r.GET("/metrics", gin.WrapH(metricsReg.Handler()))

	// Auth endpoints
	auth := r.Group("/auth")
	{
		registerHandlers := []gin.HandlerFunc{authHandler.Register}
		loginHandlers := []gin.HandlerFunc{authHandler.Login}
		forgotHandlers := []gin.HandlerFunc{authHandler.ForgotPassword}

		if cfg.RateLimit.Enabled {
			registerHandlers = append([]gin.HandlerFunc{
				middleware.RateLimit(limiter, middleware.RateLimitConfig{
					KeyFunc:      middleware.KeyByClientIP,
					Limit:        cfg.RateLimit.RegisterPerWindow,
					Window:       cfg.RateLimit.IPWindow,
					EndpointName: "register:ip",
				}, logger),
			}, registerHandlers...)

			loginHandlers = append([]gin.HandlerFunc{
				middleware.RateLimit(limiter, middleware.RateLimitConfig{
					KeyFunc:      middleware.KeyByClientIP,
					Limit:        cfg.RateLimit.LoginPerWindow,
					Window:       cfg.RateLimit.IPWindow,
					EndpointName: "login:ip",
				}, logger),
			}, loginHandlers...)

			forgotHandlers = append([]gin.HandlerFunc{
				middleware.RateLimit(limiter, middleware.RateLimitConfig{
					KeyFunc:      middleware.KeyByClientIP,
					Limit:        cfg.RateLimit.ForgotIPPerWindow,
					Window:       cfg.RateLimit.IPWindow,
					EndpointName: "forgot:ip",
				}, logger),
			}, forgotHandlers...)
		}

		auth.POST("/register", registerHandlers...)
		auth.POST("/verify-email", authHandler.VerifyEmail) // query param: ?token=
		auth.POST("/resend-verification", authHandler.ResendVerification)
		auth.POST("/login", loginHandlers...)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/forgot-password", forgotHandlers...)
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
