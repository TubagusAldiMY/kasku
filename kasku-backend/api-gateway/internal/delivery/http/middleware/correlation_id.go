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

// maxCorrelationIDLen membatasi panjang correlation ID dari klien agar tidak dipakai
// untuk log flooding / injection payload.
const maxCorrelationIDLen = 64

// CorrelationID meng-inject X-Correlation-ID ke setiap request.
// Nilai dari klien HANYA dipertahankan jika valid (charset & panjang terbatas) —
// mencegah injeksi karakter berbahaya ke log/trace. Jika tidak valid atau absen, generate baru.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(CorrelationIDHeader)
		if !isValidCorrelationID(correlationID) {
			correlationID = uuid.New().String()
		}
		c.Set(CorrelationIDKey, correlationID)
		c.Header(CorrelationIDHeader, correlationID)
		c.Next()
	}
}

// isValidCorrelationID menerima hanya [A-Za-z0-9-] dengan panjang 1..maxCorrelationIDLen.
func isValidCorrelationID(s string) bool {
	if s == "" || len(s) > maxCorrelationIDLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return false
	}
	return true
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
