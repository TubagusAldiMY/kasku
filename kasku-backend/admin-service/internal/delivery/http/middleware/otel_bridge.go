package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// BridgeToOTel enriches the active OTel span (injected by otelgin) dengan
// atribut kasku-specific dan mengekspos trace_id ke gin context agar bisa
// disertakan dalam response header atau log.
//
// Middleware ini HARUS dipasang SETELAH otelgin.Middleware() di router.
func BridgeToOTel() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span.IsRecording() {
			// Sisipkan correlation_id jika tersedia di header atau context.
			if corrID := c.GetHeader("X-Correlation-ID"); corrID != "" {
				span.SetAttributes(attribute.String("kasku.correlation_id", corrID))
			}
		}

		// Ekspos trace_id ke gin context agar bisa dipakai di log / response header.
		if sc := span.SpanContext(); sc.IsValid() {
			c.Set("trace_id", sc.TraceID().String())
		}

		c.Next()
	}
}
