package middleware

import (
	"net/http"
	"strings"
	"time"

	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/jwt"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/redis"
	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyAdminID menyimpan UUID admin (string) hasil verifikasi JWT.
	ContextKeyAdminID    = "admin_id"
	ContextKeyAdminRole  = "admin_role"
	ContextKeyAdminJTI   = "admin_jti"
	ContextKeyAdminExp   = "admin_exp"
)

// AdminAuth memvalidasi Bearer token admin (HS256) + cek blacklist Redis.
// Inject admin context (id, role, jti, exp) supaya handler tidak perlu parse ulang.
func AdminAuth(signer *jwt.Signer, blacklist *redis.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			abortWithError(c, http.StatusUnauthorized, domainerrors.ErrUnauthorized)
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenStr == "" {
			abortWithError(c, http.StatusUnauthorized, domainerrors.ErrUnauthorized)
			return
		}

		claims, err := signer.Verify(tokenStr)
		if err != nil {
			abortWithError(c, http.StatusUnauthorized, domainerrors.ErrInvalidToken)
			return
		}

		if claims.ID != "" && blacklist != nil {
			ok, err := blacklist.IsBlacklisted(c.Request.Context(), claims.ID)
			if err != nil {
				abortWithError(c, http.StatusInternalServerError, domainerrors.ErrInternal)
				return
			}
			if ok {
				abortWithError(c, http.StatusUnauthorized, domainerrors.ErrTokenRevoked)
				return
			}
		}

		c.Set(ContextKeyAdminID, claims.Subject)
		c.Set(ContextKeyAdminRole, claims.Role)
		c.Set(ContextKeyAdminJTI, claims.ID)
		if claims.ExpiresAt != nil {
			c.Set(ContextKeyAdminExp, claims.ExpiresAt.Time)
		} else {
			c.Set(ContextKeyAdminExp, time.Time{})
		}
		c.Next()
	}
}

// abortWithError menulis response error standar lalu menghentikan request.
func abortWithError(c *gin.Context, status int, err *domainerrors.DomainError) {
	c.AbortWithStatusJSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
		},
	})
}
