package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const CorrelationIDHeader = "X-Correlation-ID"

// CorrelationID meng-inject correlation ID dari upstream atau generate UUID baru.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Set("correlation_id", correlationID)
		c.Header(CorrelationIDHeader, correlationID)
		c.Next()
	}
}

// GetCorrelationID mengambil correlation ID dari Gin context.
func GetCorrelationID(c *gin.Context) string {
	if val, ok := c.Get("correlation_id"); ok {
		return val.(string)
	}
	return ""
}

// BridgeToOTel menjembatani correlation ID ke OTel span attributes dan
// meng-expose trace_id ke Gin context agar bisa disertakan dalam log/response.
// Setelah handler selesai, meng-inject X-Trace-ID response header
// agar client bisa menyalin trace ID ke Jaeger/Tempo UI untuk debugging.
// Harus dipasang SETELAH otelgin.Middleware dan CorrelationID().
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
