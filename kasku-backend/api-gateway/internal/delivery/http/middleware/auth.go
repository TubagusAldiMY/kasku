package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	grpcinfra "github.com/TubagusAldiMY/kasku/api-gateway/internal/infrastructure/grpc"
	"github.com/TubagusAldiMY/kasku/api-gateway/internal/usecase"
)

const (
	// Header keys yang diinject ke downstream service
	HeaderUserID           = "X-User-ID"
	HeaderTenantSchema     = "X-Tenant-Schema"
	HeaderSubscriptionTier = "X-Subscription-Tier"

	// Tier limit headers
	HeaderTierMaxTransactions = "X-Tier-Max-Transactions"
	HeaderTierMaxAccounts     = "X-Tier-Max-Accounts"
	HeaderTierMaxInvestments  = "X-Tier-Max-Investments"
	HeaderTierHistoryMonths   = "X-Tier-History-Months"
	HeaderTierExportCSV       = "X-Tier-Export-CSV"

	// Context key untuk parsed token
	ContextKeyParsedToken = "parsed_token"
)

// JWTVerifier adalah interface untuk verifikasi JWT.
type JWTVerifier interface {
	Verify(ctx context.Context, tokenString string) (*usecase.ParsedToken, error)
}

// TierLimitsProvider adalah interface untuk mengambil tier limits dari billing-service.
type TierLimitsProvider interface {
	GetTierLimits(ctx context.Context, userID string) grpcinfra.TierLimits
}

// Auth adalah middleware yang memverifikasi JWT dan meng-inject headers ke downstream.
func Auth(verifier JWTVerifier, tierProvider TierLimitsProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := extractBearerToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Token autentikasi tidak ditemukan atau tidak valid.",
				},
			})
			return
		}

		parsedToken, err := verifier.Verify(c.Request.Context(), tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Token tidak valid atau sudah kadaluarsa.",
				},
			})
			return
		}

		// Simpan parsed token ke context untuk digunakan middleware lain
		c.Set(ContextKeyParsedToken, parsedToken)

		userIDStr := parsedToken.UserID.String()

		// Inject identity headers
		c.Request.Header.Set(HeaderUserID, userIDStr)
		c.Request.Header.Set(HeaderTenantSchema, parsedToken.TenantSchema)
		c.Request.Header.Set(HeaderSubscriptionTier, parsedToken.SubscriptionTier)

		// Ambil tier limits dari billing-service (dengan fallback FREE tier)
		limits := tierProvider.GetTierLimits(c.Request.Context(), userIDStr)
		c.Request.Header.Set(HeaderTierMaxTransactions, fmt.Sprintf("%d", limits.MaxTransactionsPerMonth))
		c.Request.Header.Set(HeaderTierMaxAccounts, fmt.Sprintf("%d", limits.MaxFinancialAccounts))
		c.Request.Header.Set(HeaderTierMaxInvestments, fmt.Sprintf("%d", limits.MaxInvestmentInstruments))
		c.Request.Header.Set(HeaderTierHistoryMonths, fmt.Sprintf("%d", limits.HistoryRetentionMonths))
		if limits.ExportCsvEnabled {
			c.Request.Header.Set(HeaderTierExportCSV, "true")
		} else {
			c.Request.Header.Set(HeaderTierExportCSV, "false")
		}

		c.Next()
	}
}

// GetParsedToken mengambil ParsedToken dari Gin context (sudah di-set oleh Auth middleware).
func GetParsedToken(c *gin.Context) (*usecase.ParsedToken, bool) {
	val, exists := c.Get(ContextKeyParsedToken)
	if !exists {
		return nil, false
	}
	token, ok := val.(*usecase.ParsedToken)
	return token, ok
}

// extractBearerToken mengekstrak token dari Authorization: Bearer <token> header.
func extractBearerToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization header tidak ditemukan")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("format Authorization header tidak valid")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("bearer token kosong")
	}

	return token, nil
}

// RemainingTTL menghitung sisa waktu hingga token expired, minimum 0.
func RemainingTTL(expiresAt time.Time) time.Duration {
	remaining := time.Until(expiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}
