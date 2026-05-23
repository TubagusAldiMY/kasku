package http

import (
	"github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func NewRouter(h *handler.TransactionHandler, isDev bool, metricsReg *metrics.Registry, log zerolog.Logger) *gin.Engine {
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
		txs := v1.Group("/transactions")
		{
			txs.GET("", h.ListTransactions)
			txs.POST("", h.CreateTransaction)
			txs.GET("/export", h.ExportCSV)
			txs.GET("/:id", h.GetTransaction)
			txs.PUT("/:id", h.UpdateTransaction)
			txs.DELETE("/:id", h.DeleteTransaction)
		}
		cats := v1.Group("/categories")
		{
			cats.GET("", h.ListCategories)
			cats.POST("", h.CreateCategory)
			cats.PUT("/:id", h.UpdateCategory)
			cats.DELETE("/:id", h.DeleteCategory)
		}
		budgets := v1.Group("/budgets")
		{
			budgets.GET("", h.ListBudgets)
			budgets.POST("", h.CreateBudget)
			budgets.GET("/:id", h.GetBudget)
			budgets.PUT("/:id", h.UpdateBudget)
			budgets.DELETE("/:id", h.DeleteBudget)
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
