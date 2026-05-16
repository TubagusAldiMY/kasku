package http

import (
	"net/http"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/jwt"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/redis"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// RouterDeps mengumpulkan semua dependency yang dipakai router.
// Memudahkan DI tanpa positional args yang panjang.
type RouterDeps struct {
	IsDev            bool
	Logger           zerolog.Logger
	HealthHandler    *handler.HealthHandler
	AuthHandler      *handler.AuthHandler
	UserHandler      *handler.UserHandler
	SubHandler       *handler.SubscriptionHandler
	PaymentHandler   *handler.PaymentHandler
	StatsHandler     *handler.StatsHandler
	AuditLogHandler  *handler.AuditLogHandler
	JWTSigner        *jwt.Signer
	TokenBlacklist   *redis.TokenBlacklist
}

// NewRouter merangkai semua route + middleware admin-service.
//
//   Public:                /health, /metrics, /v1/admin/auth/login
//   Authenticated admin:   /v1/admin/auth/{logout,me}, /v1/admin/users/**,
//                          /v1/admin/payments, /v1/admin/stats/**,
//                          /v1/admin/audit-log
func NewRouter(deps RouterDeps) *gin.Engine {
	if !deps.IsDev {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(securityHeaders())
	r.Use(requestLogger(deps.Logger))

	r.GET("/health", deps.HealthHandler.Health)
	r.GET("/metrics", metrics("admin-service"))

	// Public admin endpoints (no JWT required)
	publicAdmin := r.Group("/v1/admin")
	{
		publicAdmin.POST("/auth/login", deps.AuthHandler.Login)
	}

	// Authenticated admin endpoints (HS256 admin JWT required)
	authMW := middleware.AdminAuth(deps.JWTSigner, deps.TokenBlacklist)
	protected := r.Group("/v1/admin")
	protected.Use(authMW)
	{
		protected.POST("/auth/logout", deps.AuthHandler.Logout)
		protected.GET("/auth/me", deps.AuthHandler.Me)

		protected.GET("/users", deps.UserHandler.List)
		protected.GET("/users/:id", deps.UserHandler.Detail)
		protected.POST("/users/:id/suspend", deps.UserHandler.Suspend)
		protected.POST("/users/:id/activate", deps.UserHandler.Activate)
		protected.POST("/users/:id/override-subscription", deps.SubHandler.Override)

		protected.GET("/payments", deps.PaymentHandler.List)
		protected.GET("/stats/dashboard", deps.StatsHandler.Dashboard)
		protected.GET("/audit-log", deps.AuditLogHandler.List)
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
