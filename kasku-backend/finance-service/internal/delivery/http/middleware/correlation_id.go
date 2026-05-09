package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const correlationIDKey = "correlation_id"

func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Correlation-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Header("X-Correlation-ID", id)
		c.Set(correlationIDKey, id)
		c.Next()
	}
}
