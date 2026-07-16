package http

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

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
	// TrustedProxies berisi CIDR proxy yang dipercaya untuk header X-Forwarded-For.
	// Kosong = tidak percaya proxy apa pun (ClientIP jatuh ke direct peer/Traefik).
	TrustedProxies []string
	// MetricsToken menggate GET /metrics via bearer token. Kosong = /metrics dinonaktifkan.
	MetricsToken string
}

// NewRouter membuat dan mengkonfigurasi Gin router dengan semua middleware dan route.
func NewRouter(cfg RouterConfig) *gin.Engine {
	if !cfg.IsDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Trusted proxies — WAJIB agar c.ClientIP() tidak percaya X-Forwarded-For dari klien.
	// Jika kosong, SetTrustedProxies(nil) membuat ClientIP jatuh ke direct peer (Traefik),
	// sehingga XFF yang di-spoof diabaikan dan rate limit per-IP tidak bisa di-bypass.
	// Operator HARUS set GATEWAY_TRUSTED_PROXIES ke subnet CIDR Traefik untuk memulihkan IP klien asli.
	// SetTrustedProxies hanya error jika CIDR tidak valid — fatal saat startup agar misconfig ketahuan.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		cfg.Logger.Fatal().Err(err).Msg("GATEWAY_TRUSTED_PROXIES tidak valid")
	}

	// Recovery — mencegah crash dari panic
	r.Use(gin.Recovery())

	// Strip identity headers yang diinject gateway dari SEMUA request masuk,
	// sebelum route group mana pun. Membuat spoofing identitas mustahil secara struktural.
	r.Use(stripInjectedIdentityHeaders())

	// CORS — harus sebelum route lain agar OPTIONS preflight bisa dihandle
	r.Use(cfg.CORSMiddleware)

	// Correlation ID — inject ke setiap request
	r.Use(middleware.CorrelationID())

	// Prometheus HTTP metrics (sebelum route apa pun)
	r.Use(cfg.Metrics.HTTPMetrics())

	// OTel distributed tracing — setelah metrics, sebelum security headers
	r.Use(otelgin.Middleware("api-gateway"))
	r.Use(middleware.BridgeToOTel())

	// Security headers (OWASP)
	r.Use(securityHeaders())

	// Request logger
	r.Use(requestLogger(cfg.Logger))

	// Health check (public, tanpa auth)
	r.GET("/health", cfg.HealthHandler.Health)

	// /metrics — hanya diaktifkan jika METRICS_TOKEN diset, dan digate via bearer token.
	// Tanpa token, endpoint tidak didaftarkan sama sekali (tidak ada exposure Prometheus tanpa auth).
	if cfg.MetricsToken != "" {
		r.GET("/metrics", metricsAuth(cfg.MetricsToken), gin.WrapH(cfg.Metrics.Handler()))
	} else {
		cfg.Logger.Warn().Msg("METRICS_TOKEN kosong — endpoint /metrics dinonaktifkan")
	}

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
		v1Auth.POST("/google", cfg.ProxyHandler.ProxyTo("auth"))
		v1Auth.POST("/google/code", cfg.ProxyHandler.ProxyTo("auth"))
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

	// Payment webhook: skip JWT auth, verifikasi HMAC-SHA256 dilakukan di billing-service.
	r.POST("/v1/billing/webhook/payment", cfg.ProxyHandler.ProxyTo("billing"))

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

	// ── /v1/budgets/** ────────────────────────────────────────────────────────
	v1Budgets := r.Group("/v1/budgets")
	v1Budgets.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Budgets.Any("", cfg.ProxyHandler.ProxyTo("transaction"))
		v1Budgets.Any("/*path", cfg.ProxyHandler.ProxyTo("transaction"))
	}

	// ── /v1/investments/** ────────────────────────────────────────────────
	v1Investments := r.Group("/v1/investments")
	v1Investments.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Investments.Any("", cfg.ProxyHandler.ProxyTo("investment"))
		v1Investments.Any("/*path", cfg.ProxyHandler.ProxyTo("investment"))
	}

	// ── /v1/debts/** ────────────────────────────────────────────────────
	v1Debts := r.Group("/v1/debts")
	v1Debts.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
	{
		v1Debts.Any("", cfg.ProxyHandler.ProxyTo("finance"))
		v1Debts.Any("/*path", cfg.ProxyHandler.ProxyTo("finance"))
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

// injectedIdentityHeaders adalah semua header identitas yang HANYA boleh diset oleh
// gateway (Auth middleware). Request masuk tidak boleh membawanya — kalau ada, itu spoofing.
var injectedIdentityHeaders = []string{
	middleware.HeaderUserID,
	middleware.HeaderUserEmail,
	middleware.HeaderTenantSchema,
	middleware.HeaderSubscriptionTier,
	middleware.HeaderTierMaxTransactions,
	middleware.HeaderTierMaxAccounts,
	middleware.HeaderTierMaxInvestments,
	middleware.HeaderTierHistoryMonths,
	middleware.HeaderTierExportCSV,
}

// stripInjectedIdentityHeaders menghapus semua header identitas yang diinject gateway
// dari request masuk, tanpa syarat. Dijalankan sebelum route group mana pun sehingga
// tidak ada rute yang bisa menerima identitas hasil spoofing dari klien.
func stripInjectedIdentityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, h := range injectedIdentityHeaders {
			c.Request.Header.Del(h)
		}
		c.Next()
	}
}

// metricsAuth menggate endpoint /metrics via bearer token dengan perbandingan constant-time.
func metricsAuth(token string) gin.HandlerFunc {
	expected := []byte("Bearer " + token)
	return func(c *gin.Context) {
		got := []byte(c.GetHeader("Authorization"))
		if len(got) != len(expected) || subtle.ConstantTimeCompare(got, expected) != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

// securityHeaders meng-inject OWASP security headers ke setiap response.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		// HSTS hanya dikirim saat request datang via HTTPS (lewat Traefik TLS termination
		// yang meneruskan header X-Forwarded-Proto).
		if c.GetHeader("X-Forwarded-Proto") == "https" || c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}
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
			Str("trace_id", c.GetString("trace_id")).
			Str("client_ip", c.ClientIP()).
			Msg("http request")
	}
}
