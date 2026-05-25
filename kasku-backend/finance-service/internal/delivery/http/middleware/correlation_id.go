package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

// GetCorrelationID mengambil correlation ID dari gin context.
func GetCorrelationID(c *gin.Context) string {
	if v, ok := c.Get(correlationIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// BridgeToOTel menjembatani correlation ID ke OTel span dan
// menyimpan trace ID ke gin context agar dapat disertakan di response.
// Harus dipasang SETELAH otelgin.Middleware dan CorrelationID.
func BridgeToOTel() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			if corrID := GetCorrelationID(c); corrID != "" {
				span.SetAttributes(attribute.String("kasku.correlation_id", corrID))
			}
		}
		if sc := span.SpanContext(); sc.IsValid() {
			c.Set("trace_id", sc.TraceID().String())
		}
		c.Next()
	}
}
