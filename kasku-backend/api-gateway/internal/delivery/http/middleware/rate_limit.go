package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/TubagusAldiMY/kasku/api-gateway/internal/usecase"
)

// maxRateLimitBodyPeek membatasi berapa banyak body yang dibaca saat mengekstrak email
// untuk keperluan rate limiting. Body lengkap tetap di-restore ke request; batas ini
// hanya melindungi dari body raksasa sambil tetap menampung payload login normal.
const maxRateLimitBodyPeek = 1 << 20 // 1 MiB

// RateLimiter adalah interface untuk use case rate limiting.
type RateLimiter interface {
	CheckRegister(ctx context.Context, clientIP string) (*usecase.RateLimitCheckResult, error)
	CheckLogin(ctx context.Context, clientIP, email string) (*usecase.RateLimitCheckResult, error)
	CheckRefresh(ctx context.Context, userID string) (*usecase.RateLimitCheckResult, error)
	CheckForgotPassword(ctx context.Context, email string) (*usecase.RateLimitCheckResult, error)
	CheckDefault(ctx context.Context, userID string) (*usecase.RateLimitCheckResult, error)
	CheckSync(ctx context.Context, userID string) (*usecase.RateLimitCheckResult, error)
}

// RateLimit adalah middleware yang menerapkan rate limiting per endpoint.
func RateLimit(limiter RateLimiter, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		method := c.Request.Method
		clientIP := c.ClientIP()

		var result *usecase.RateLimitCheckResult
		var err error
		// authSensitive menandai jalur credential (register/login/refresh/forgot-password).
		// Untuk jalur ini limiter WAJIB fail-closed jika error (jangan buka pintu brute-force).
		authSensitive := false

		switch {
		case method == http.MethodPost && strings.HasSuffix(path, "/auth/register"):
			authSensitive = true
			result, err = limiter.CheckRegister(c.Request.Context(), clientIP)

		case method == http.MethodPost && strings.HasSuffix(path, "/auth/login"):
			authSensitive = true
			email := extractEmailFromBody(c)
			result, err = limiter.CheckLogin(c.Request.Context(), clientIP, email)

		case method == http.MethodPost && strings.HasSuffix(path, "/auth/refresh"):
			authSensitive = true
			userID := ""
			if token, ok := GetParsedToken(c); ok {
				userID = token.UserID.String()
			}
			if userID == "" {
				// Refresh tidak melewati auth middleware, jadi biasanya belum ada parsed token.
				// Pakai bucket refresh berbasis IP agar tidak menghabiskan limit register.
				result, err = limiter.CheckRefresh(c.Request.Context(), "ip:"+clientIP)
			} else {
				result, err = limiter.CheckRefresh(c.Request.Context(), userID)
			}

		case method == http.MethodPost && strings.HasSuffix(path, "/auth/forgot-password"):
			authSensitive = true
			email := extractEmailFromBody(c)
			result, err = limiter.CheckForgotPassword(c.Request.Context(), email)

		case strings.Contains(path, "/sync/"):
			if token, ok := GetParsedToken(c); ok {
				result, err = limiter.CheckSync(c.Request.Context(), token.UserID.String())
			} else {
				c.Next()
				return
			}

		default:
			// Default rate limit — butuh user ID dari parsed token
			if token, ok := GetParsedToken(c); ok {
				result, err = limiter.CheckDefault(c.Request.Context(), token.UserID.String())
			} else {
				// Endpoint publik tanpa token — tidak perlu rate limit di sini
				c.Next()
				return
			}
		}

		if err != nil {
			// OWASP A09 — jangan pernah menelan error limiter secara diam-diam.
			logger.Error().
				Err(err).
				Str("path", path).
				Str("method", method).
				Str("client_ip", clientIP).
				Bool("auth_sensitive", authSensitive).
				Msg("rate limiter error")

			if authSensitive {
				// Fail-CLOSED pada jalur credential: tolak request agar brute-force tidak lolos
				// saat backend limiter (Redis) bermasalah.
				c.Header("Retry-After", "5")
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "RATE_LIMITER_UNAVAILABLE",
						"message": "Layanan sedang sibuk. Coba lagi sebentar.",
					},
				})
				return
			}
			// Fail-open HANYA untuk bucket default terautentikasi (ketersediaan diutamakan).
			c.Next()
			return
		}

		// Inject rate limit response headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
		remaining := result.Limit - result.Current
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetTime.Unix()))

		if !result.Allowed {
			retryAfter := int(time.Until(result.ResetTime).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Terlalu banyak request. Coba lagi nanti.",
				},
			})
			return
		}

		c.Next()
	}
}

// extractEmailFromBody membaca field "email" dari JSON request body.
// Body LENGKAP dibaca (dibatasi maxRateLimitBodyPeek) lalu di-restore utuh ke request,
// sehingga handler downstream / proxy tetap menerima body yang tidak terpotong.
func extractEmailFromBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxRateLimitBodyPeek))
	if err != nil {
		return ""
	}
	// Restore body LENGKAP untuk handler downstream sebelum parsing apa pun.
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return ""
	}
	return body.Email
}
