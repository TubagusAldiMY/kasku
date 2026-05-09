package http

import (
	"github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// NewRouter membuat Gin router dengan semua middleware dan route yang terkonfigurasi.
// Mode Gin (debug/release) ditentukan berdasarkan environment.
func NewRouter(billingHandler *handler.BillingHandler, isDev bool, log zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())
	r.Use(securityHeadersMiddleware())

	// Health check tidak memerlukan autentikasi — diakses oleh load balancer dan Docker healthcheck
	r.GET("/health", billingHandler.Health)

	v1 := r.Group("/v1")
	{
		billing := v1.Group("/billing")
		{
			// Public endpoint — tidak memerlukan JWT (pricing page)
			billing.GET("/plans", billingHandler.ListPlans)

			// Protected endpoint — memerlukan X-User-ID header dari api-gateway
			billing.GET("/subscription", billingHandler.GetSubscription)
			billing.POST("/subscribe", billingHandler.Subscribe)

			// Midtrans webhook — tidak ada JWT, verifikasi via HMAC signature (Phase 2)
			billing.POST("/webhook/midtrans", billingHandler.MidtransWebhook)
		}
	}

	return r
}

// securityHeadersMiddleware menambahkan security headers standar OWASP ke semua response.
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
