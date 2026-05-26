package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const CorrelationIDHeader = "X-Correlation-ID"

func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Header(CorrelationIDHeader, correlationID)
		c.Set("correlation_id", correlationID)
		c.Next()
	}
}

// GetCorrelationID mengambil correlation ID dari Gin context.
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get("correlation_id"); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// BridgeToOTel menjembatani correlation ID ke span OTel aktif dan menyimpan
// trace_id ke Gin context agar bisa di-log oleh request logger.
// Setelah handler selesai, meng-inject X-Trace-ID response header
// agar client bisa menyalin trace ID ke Jaeger/Tempo UI untuk debugging.
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
			traceID := sc.TraceID().String()
			c.Set("trace_id", traceID)
			// X-Trace-ID dikirim ke client hanya jika trace aktif (bukan zero value),
			// sehingga developer bisa copy-paste langsung ke Jaeger/Tempo UI.
			c.Header("X-Trace-ID", traceID)
		}
		c.Next()
	}
}
