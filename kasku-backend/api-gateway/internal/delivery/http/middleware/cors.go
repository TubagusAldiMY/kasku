package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS menangani Cross-Origin Resource Sharing berdasarkan daftar origin yang diizinkan.
// Jika origin request tidak ada dalam allowedOrigins, request di-reject dengan 403.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowedSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin == "" {
			// Bukan cross-origin request — lanjutkan tanpa CORS headers
			c.Next()
			return
		}

		_, allowed := allowedSet[origin]
		if !allowed {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Correlation-ID,X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "X-Correlation-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset")
		c.Header("Access-Control-Max-Age", "86400")

		// Preflight request
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
