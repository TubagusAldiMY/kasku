package http

import (
	"net/http"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
)

func NewRouter(h *handler.AccountHandler, isDev bool) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())
	r.Use(securityHeaders())

	r.GET("/health", h.Health)
	r.GET("/metrics", metrics("finance-service"))

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

func metrics(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(
			"# HELP kasku_service_info KasKu service metadata\n"+
				"# TYPE kasku_service_info gauge\n"+
				"kasku_service_info{service=\""+service+"\"} 1\n",
		))
	}
}

// securityHeaders menambahkan security headers standar OWASP pada semua response.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
