package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const correlationIDHeader = "X-Correlation-ID"

// CorrelationID adalah middleware yang memastikan setiap request memiliki correlation ID
// untuk keperluan distributed tracing. Jika header tidak ada, UUID baru di-generate.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(correlationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Header(correlationIDHeader, correlationID)
		c.Set("correlation_id", correlationID)
		c.Next()
	}
}
