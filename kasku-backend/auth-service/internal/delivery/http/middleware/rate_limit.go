package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// RateLimitConfig adalah konfigurasi untuk satu endpoint rate-limit middleware.
type RateLimitConfig struct {
	// KeyFunc mengekstrak bagian unik dari context (mis. ClientIP atau hash email).
	// Akan digabung dengan EndpointName untuk membentuk key Redis.
	KeyFunc func(c *gin.Context) string

	// Limit adalah jumlah maksimum request dalam Window.
	Limit int

	// Window adalah panjang sliding window.
	Window time.Duration

	// EndpointName dipakai sebagai bagian prefix Redis key (mis. "login:ip").
	EndpointName string
}

// RateLimit membuat middleware Gin untuk rate-limit berbasis sliding window log.
//
// Response 429 mengandung:
//   - Header Retry-After (detik)
//   - Header X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
//   - Body JSON {"success":false,"error":{"code":"TOO_MANY_REQUESTS","message":"..."}}
//
// Jika Redis tidak responsif, middleware fail-open (allow request) dan log warning —
// availability service lebih penting dari rate-limit enforcement saat infra error.
func RateLimit(limiter ratelimit.Limiter, cfg RateLimitConfig, log zerolog.Logger) gin.HandlerFunc {
	if cfg.Limit <= 0 {
		panic("rate limit: Limit harus > 0")
	}
	if cfg.Window <= 0 {
		panic("rate limit: Window harus > 0")
	}
	if cfg.KeyFunc == nil {
		panic("rate limit: KeyFunc wajib di-set")
	}
	if cfg.EndpointName == "" {
		panic("rate limit: EndpointName wajib di-set")
	}

	return func(c *gin.Context) {
		identity := cfg.KeyFunc(c)
		if identity == "" {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", cfg.EndpointName, identity)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		retryAfter, err := limiter.Check(ctx, key, cfg.Limit, cfg.Window)
		if err != nil {
			if errors.Is(err, ratelimit.ErrLimitExceeded) {
				retryAfterSec := max(int64(retryAfter.Seconds()), 1)
				resetUnix := time.Now().Unix() + retryAfterSec

				c.Header("Retry-After", strconv.FormatInt(retryAfterSec, 10))
				c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", strconv.FormatInt(resetUnix, 10))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "TOO_MANY_REQUESTS",
						"message": fmt.Sprintf("Terlalu banyak permintaan. Silakan coba lagi dalam %d detik.", retryAfterSec),
					},
				})
				return
			}

			// Fail-open: Redis error atau response invalid → izinkan request, log warning
			log.Warn().Err(err).
				Str("endpoint", cfg.EndpointName).
				Msg("rate-limit fail-open")
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
		c.Next()
	}
}

// KeyByClientIP adalah KeyFunc bawaan yang memakai Gin ClientIP().
// Pastikan engine.SetTrustedProxies sudah dikonfigurasi dengan benar di production.
func KeyByClientIP(c *gin.Context) string {
	return c.ClientIP()
}
