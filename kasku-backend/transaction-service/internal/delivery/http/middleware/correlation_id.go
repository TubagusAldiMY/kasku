package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// CorrelationIDHeader adalah nama HTTP header untuk correlation ID.
	CorrelationIDHeader = "X-Correlation-ID"
	// correlationIDKey adalah key yang digunakan di Gin context.
	correlationIDKey = "correlation_id"
)

// CorrelationID adalah middleware yang memastikan setiap request memiliki correlation ID.
// Jika header tidak ada, UUID baru di-generate.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(CorrelationIDHeader)
		if id == "" {
			id = uuid.New().String()
		}
		c.Header(CorrelationIDHeader, id)
		c.Set(correlationIDKey, id)
		c.Next()
	}
}

// GetCorrelationID mengambil correlation ID dari Gin context.
// Mengembalikan "" jika tidak ditemukan.
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get(correlationIDKey); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// BridgeToOTel menjembatani correlation ID ke active OTel span dan
// meng-inject trace_id ke Gin context untuk keperluan logging.
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
