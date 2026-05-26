package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	CorrelationIDHeader = "X-Correlation-ID"
	CorrelationIDKey    = "correlation_id"
)

// CorrelationID meng-inject X-Correlation-ID ke setiap request.
// Jika header sudah ada dari upstream client, nilainya dipertahankan.
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

// BridgeToOTel menyalin correlation_id ke active OTel span dan
// menginjeksi trace_id ke gin.Context untuk requestLogger.
// Setelah handler selesai, meng-inject X-Trace-ID response header
// agar client bisa menyalin trace ID ke Jaeger/Tempo UI untuk debugging.
// Harus dipanggil SETELAH otelgin.Middleware().
func BridgeToOTel() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			if corrID := GetCorrelationID(c); corrID != "" {
				span.SetAttributes(attribute.String("kasku.correlation_id", corrID))
			}
		}
		if sc := span.SpanContext(); sc.IsValid() {
			traceID := sc.TraceID().String()
			c.Set("trace_id", traceID)
			// X-Trace-ID dikirim ke client hanya jika trace aktif (bukan zero value),
			// sehingga developer bisa copy-paste langsung ke Jaeger/Tempo UI.
			c.Header("X-Trace-ID", traceID)
		}
		c.Next()
	}
}
