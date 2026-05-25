package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	CorrelationIDHeader = "X-Correlation-ID"
	CorrelationIDKey    = "correlation_id"
)

// correlationIDContextKey adalah private type untuk menghindari tabrakan key
// di context.Value. Jangan ekspor — gunakan helper di bawah.
type correlationIDContextKey struct{}

// ContextWithCorrelationID menempelkan correlation ID ke context.Context standar.
// Dipakai oleh gRPC interceptor (yang tidak memiliki gin.Context).
func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDContextKey{}, id)
}

// CorrelationIDFromContext mengambil correlation ID dari context.Context standar.
// Mengembalikan "" jika tidak ada.
func CorrelationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(correlationIDContextKey{}).(string); ok {
		return v
	}
	return ""
}

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

// BridgeToOTel menghubungkan correlation_id ke span OTel yang aktif dan menyimpan
// trace_id ke Gin context agar bisa disertakan di response log.
// Harus dipasang SETELAH otelgin.Middleware (agar span sudah ada) dan SETELAH CorrelationID().
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
