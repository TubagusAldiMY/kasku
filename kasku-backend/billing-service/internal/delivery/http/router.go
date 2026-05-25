package http

import (
	"github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// NewRouter membuat Gin router dengan semua middleware dan route yang terkonfigurasi.
func NewRouter(billingHandler *handler.BillingHandler, isDev bool, metricsReg *metrics.Registry, log zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("billing-service"))
	r.Use(middleware.CorrelationID())
	r.Use(middleware.BridgeToOTel())
	r.Use(metricsReg.HTTPMetrics())
	r.Use(securityHeadersMiddleware())

	r.GET("/health", billingHandler.Health)
	r.GET("/metrics", gin.WrapH(metricsReg.Handler()))

	v1 := r.Group("/v1")
	{
		billing := v1.Group("/billing")
		{
			billing.GET("/plans", billingHandler.ListPlans)
			billing.GET("/subscription", billingHandler.GetSubscription)
			billing.POST("/subscribe", billingHandler.Subscribe)

			// Endpoint webhook Payment Orchestrator — tidak memerlukan JWT.
			// Keamanan dijamin via HMAC-SHA256 signature verification di handler.
			billing.POST("/webhook/payment", billingHandler.PaymentWebhook)
		}
	}

	return r
}

func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
