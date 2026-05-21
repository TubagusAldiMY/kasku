package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/TubagusAldiMY/kasku/api-gateway/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/api-gateway/internal/delivery/http/middleware"
	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
)

// RouterConfig menyimpan semua dependency untuk membuat router.
type RouterConfig struct {
	HealthHandler       *handler.HealthHandler
	ProxyHandler        *handler.ProxyHandler
	AuthMiddleware      gin.HandlerFunc
	RateLimitMiddleware gin.HandlerFunc
	CORSMiddleware      gin.HandlerFunc
	IsDev               bool
	Logger              zerolog.Logger
	Metrics             *obsmetrics.Registry
}

// NewRouter membuat dan mengkonfigurasi Gin router dengan semua middleware dan route.
func NewRouter(cfg RouterConfig) *gin.Engine {
	if !cfg.IsDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Recovery — mencegah crash dari panic
	r.Use(gin.Recovery())

	// CORS — harus sebelum route lain agar OPTIONS preflight bisa dihandle
	r.Use(cfg.CORSMiddleware)

	// Correlation ID — inject ke setiap request
	r.Use(middleware.CorrelationID())

	// Prometheus HTTP metrics (sebelum route apa pun)
	r.Use(cfg.Metrics.HTTPMetrics())

	// Security headers (OWASP)
	r.Use(securityHeaders())

	// Request logger
	r.Use(requestLogger(cfg.Logger))

	// Health check (public, tanpa auth)
	r.GET("/health", cfg.HealthHandler.Health)
	r.GET("/metrics", gin.WrapH(cfg.Metrics.Handler()))

	// ── /v1/auth/** ───────────────────────────────────────────────────────────
	// Catatan: sebagian besar auth endpoint adalah public (tidak butuh JWT),
	// tapi tetap butuh rate limiting.
	v1Auth := r.Group("/v1/auth")
	v1Auth.Use(cfg.RateLimitMiddleware)
	{
		v1Auth.POST("/register", cfg.ProxyHandler.ProxyTo("auth"))
		v1Auth.POST("/verify-email", cfg.ProxyHandler.ProxyTo("auth"))
		v1Auth.POST("/resend-verification", cfg.ProxyHandler.ProxyTo("auth"))
		v1Auth.POST("/login", cfg.ProxyHandler.ProxyTo("auth"))
		v1Auth.POST("/refresh", cfg.ProxyHandler.ProxyTo("auth"))
		v1Auth.POST("/forgot-password", cfg.ProxyHandler.ProxyTo("auth"))
		v1Auth.POST("/reset-password", cfg.ProxyHandler.ProxyTo("auth"))
		// Logout dan change-password butuh JWT
		v1Auth.POST("/logout", cfg.AuthMiddleware, cfg.ProxyHandler.ProxyTo("auth"))
		v1Auth.PUT("/change-password", cfg.AuthMiddleware, cfg.ProxyHandler.ProxyTo("auth"))
	}

	// ── /v1/users/** ──────────────────────────────────────────────────────────
	v1Users := r.Group("/v1/users")
	v1Users.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Users.Any("", cfg.ProxyHandler.ProxyTo("user"))
		v1Users.Any("/*path", cfg.ProxyHandler.ProxyTo("user"))
	}

	// ── /v1/billing/** ────────────────────────────────────────────────────────
	// Catatan: wildcard tidak bisa digabung dengan static route pada prefix yang sama
	// di httprouter (Gin's router). Billing endpoint terbatas dan sudah diketahui,
	// jadi didaftarkan secara eksplisit untuk Phase 1.

	// Midtrans webhook: skip JWT auth, verifikasi signature dilakukan di billing-service.
	r.POST("/v1/billing/webhook/midtrans", cfg.ProxyHandler.ProxyTo("billing"))

	// Billing endpoints yang butuh JWT
	v1Billing := r.Group("/v1/billing")
	v1Billing.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Billing.GET("/plans", cfg.ProxyHandler.ProxyTo("billing"))
		v1Billing.GET("/subscription", cfg.ProxyHandler.ProxyTo("billing"))
		v1Billing.POST("/subscribe", cfg.ProxyHandler.ProxyTo("billing"))
	}

	// ── /v1/accounts/** ───────────────────────────────────────────────────────
	v1Accounts := r.Group("/v1/accounts")
	v1Accounts.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Accounts.Any("", cfg.ProxyHandler.ProxyTo("finance"))
		v1Accounts.Any("/*path", cfg.ProxyHandler.ProxyTo("finance"))
	}

	// ── /v1/transactions/** ───────────────────────────────────────────────────
	v1Transactions := r.Group("/v1/transactions")
	v1Transactions.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Transactions.Any("", cfg.ProxyHandler.ProxyTo("transaction"))
		v1Transactions.Any("/*path", cfg.ProxyHandler.ProxyTo("transaction"))
	}

	// ── /v1/categories/** ─────────────────────────────────────────────────────
	v1Categories := r.Group("/v1/categories")
	v1Categories.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Categories.Any("", cfg.ProxyHandler.ProxyTo("transaction"))
		v1Categories.Any("/*path", cfg.ProxyHandler.ProxyTo("transaction"))
	}
	// ── /v1/investments/** ────────────────────────────────────────────────
	v1Investments := r.Group("/v1/investments")
	v1Investments.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Investments.Any("", cfg.ProxyHandler.ProxyTo("investment"))
		v1Investments.Any("/*path", cfg.ProxyHandler.ProxyTo("investment"))
	}

	// ── /v1/prices/** ───────────────────────────────────────────────────
	v1Prices := r.Group("/v1/prices")
	v1Prices.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Prices.GET("/:symbol", cfg.ProxyHandler.ProxyTo("price"))
	}

	// ── /v1/sync/** ───────────────────────────────────────────────────────
	v1Sync := r.Group("/v1/sync")
	v1Sync.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Sync.Any("", cfg.ProxyHandler.ProxyTo("sync"))
		v1Sync.Any("/*path", cfg.ProxyHandler.ProxyTo("sync"))
	}

	// ── /v1/notifications/** ────────────────────────────────────────────
	v1Notifications := r.Group("/v1/notifications")
	v1Notifications.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Notifications.Any("", cfg.ProxyHandler.ProxyTo("notification"))
		v1Notifications.Any("/*path", cfg.ProxyHandler.ProxyTo("notification"))
	}

	// ── /v1/admin/** ────────────────────────────────────────────────────
	// CATATAN: admin-service verify JWT HS256 sendiri (terpisah dari user RS256 JWT).
	// Gateway TIDAK pasang AuthMiddleware di sini — biarkan request langsung ke admin-service.
	// Tetap kena RateLimitMiddleware untuk DoS protection.
	v1Admin := r.Group("/v1/admin")
	v1Admin.Use(cfg.RateLimitMiddleware)
	{
		v1Admin.Any("", cfg.ProxyHandler.ProxyTo("admin"))
		v1Admin.Any("/*path", cfg.ProxyHandler.ProxyTo("admin"))
	}

	return r
}

// securityHeaders meng-inject OWASP security headers ke setiap response.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// requestLogger mencatat setiap HTTP request dalam format JSON terstruktur.
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
