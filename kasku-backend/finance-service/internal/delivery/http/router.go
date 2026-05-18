package http

import (
	"github.com/TubagusAldiMY/kasku/finance-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/gin-gonic/gin"
)

func NewRouter(h *handler.AccountHandler, isDev bool, metricsReg *metrics.Registry) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())
	r.Use(metricsReg.HTTPMetrics())
	r.Use(securityHeaders())

	r.GET("/health", h.Health)
	r.GET("/metrics", gin.WrapH(metricsReg.Handler()))

	v1 := r.Group("/v1")
	{
		accounts := v1.Group("/accounts")
		{
			accounts.GET("", h.ListAccounts)
			accounts.POST("", h.CreateAccount)
			accounts.GET("/:id", h.GetAccount)
			accounts.PUT("/:id", h.UpdateAccount)
			accounts.DELETE("/:id", h.DeleteAccount)
			accounts.GET("/:id/history", h.GetBalanceHistory)
		}
	}

	return r
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
