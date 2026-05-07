package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	CorrelationIDHeader = "X-Correlation-ID"
	CorrelationIDKey    = "correlation_id"
)

// CorrelationID meng-inject X-Correlation-ID ke setiap request.
// Jika header sudah ada, nilainya dipertahankan. Jika tidak, UUID baru digenerate.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Set(CorrelationIDKey, correlationID)
		c.Header(CorrelationIDHeader, correlationID)
		c.Next()
	}
}

// GetCorrelationID mengambil correlation ID dari Gin context.
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get(CorrelationIDKey); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}
