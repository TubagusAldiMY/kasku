package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/TubagusAldiMY/kasku/api-gateway/internal/usecase"
)

// RateLimiter adalah interface untuk use case rate limiting.
type RateLimiter interface {
	CheckRegister(ctx context.Context, clientIP string) (*usecase.RateLimitCheckResult, error)
	CheckLogin(ctx context.Context, clientIP, email string) (*usecase.RateLimitCheckResult, error)
	CheckRefresh(ctx context.Context, userID string) (*usecase.RateLimitCheckResult, error)
	CheckForgotPassword(ctx context.Context, email string) (*usecase.RateLimitCheckResult, error)
	CheckDefault(ctx context.Context, userID string) (*usecase.RateLimitCheckResult, error)
}

// RateLimit adalah middleware yang menerapkan rate limiting per endpoint.
func RateLimit(limiter RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		method := c.Request.Method
		clientIP := c.ClientIP()

		var result *usecase.RateLimitCheckResult
		var err error

		switch {
		case method == http.MethodPost && strings.HasSuffix(path, "/auth/register"):
			result, err = limiter.CheckRegister(c.Request.Context(), clientIP)

		case method == http.MethodPost && strings.HasSuffix(path, "/auth/login"):
			email := extractEmailFromBody(c)
			result, err = limiter.CheckLogin(c.Request.Context(), clientIP, email)

		case method == http.MethodPost && strings.HasSuffix(path, "/auth/refresh"):
			userID := ""
			if token, ok := GetParsedToken(c); ok {
				userID = token.UserID.String()
			}
			if userID == "" {
				// Refresh tidak butuh auth middleware duluan — gunakan IP sebagai fallback
				result, err = limiter.CheckRegister(c.Request.Context(), clientIP)
			} else {
				result, err = limiter.CheckRefresh(c.Request.Context(), userID)
			}

		case method == http.MethodPost && strings.HasSuffix(path, "/auth/forgot-password"):
			email := extractEmailFromBody(c)
			result, err = limiter.CheckForgotPassword(c.Request.Context(), email)

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
			// Jika rate limiter error, biarkan request lewat (fail-open untuk ketersediaan)
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

// extractEmailFromBody mencoba membaca field "email" dari JSON request body.
// Body di-cache ulang agar handler downstream tetap bisa membacanya.
func extractEmailFromBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 1024))
	if err != nil {
		return ""
	}
	// Restore body untuk handler downstream
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return ""
	}
	return body.Email
}
