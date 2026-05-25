package http

import (
	"time"

	"github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/TubagusAldiMY/kasku/user-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/user-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func NewRouter(userHandler *handler.UserHandler, isDev bool, metricsReg *metrics.Registry, log zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// OTel tracing — harus dipasang sebelum CorrelationID agar span context
	// tersedia saat BridgeToOTel membaca span.
	r.Use(otelgin.Middleware("user-service"))
	r.Use(middleware.CorrelationID())
	r.Use(middleware.BridgeToOTel())

	r.Use(metricsReg.HTTPMetrics())

	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	// Request logger — mencatat correlation_id dan trace_id untuk observability.
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		traceID, _ := c.Get("trace_id")
		corrID, _ := c.Get("correlation_id")
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency_ms", time.Since(start)).
			Str("correlation_id", safeStr(corrID)).
			Str("trace_id", safeStr(traceID)).
			Msg("request")
	})

	r.GET("/health", userHandler.Health)
	r.GET("/metrics", gin.WrapH(metricsReg.Handler()))

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

func safeStr(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
