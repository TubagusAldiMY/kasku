package http

import (
	"net/http"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func NewRouter(h *handler.TransactionHandler, isDev bool, log zerolog.Logger) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())
	r.Use(securityHeaders())

	r.GET("/health", h.Health)
	r.GET("/metrics", metrics("transaction-service"))

	v1 := r.Group("/v1")
	{
		txs := v1.Group("/transactions")
		{
			txs.GET("", h.ListTransactions)
			txs.POST("", h.CreateTransaction)
			txs.GET("/export", h.ExportCSV)
			txs.GET("/:id", h.GetTransaction)
			txs.DELETE("/:id", h.DeleteTransaction)
		}
		cats := v1.Group("/categories")
		{
			cats.GET("", h.ListCategories)
			cats.POST("", h.CreateCategory)
			cats.PUT("/:id", h.UpdateCategory)
			cats.DELETE("/:id", h.DeleteCategory)
		}
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

// securityHeaders agrega security response headers standar OWASP.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
