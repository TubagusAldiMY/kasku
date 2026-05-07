package usecase

import (
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateAccessToken membuat JWT RS256 dengan custom claims KasKu.
// Ini adalah fungsi shared yang digunakan oleh LoginUseCase dan RefreshTokenUseCase.
func GenerateAccessToken(user *entity.User, privateKey *rsa.PrivateKey, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	jti := uuid.New().String()

	// tenant_schema = "tenant_" + user_id dengan dash diganti underscore
	tenantSchema := "tenant_" + strings.ReplaceAll(user.ID.String(), "-", "_")

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Email:            user.Email,
		TenantSchema:     tenantSchema,
		SubscriptionTier: subscriptionTierFree,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("gagal sign JWT: %w", err)
	}

	return signed, nil
}

// ParseAccessToken mem-parse dan memverifikasi JWT menggunakan RSA public key.
// Mengizinkan token yang sudah expired agar logout tetap bisa memproses JTI.
func ParseAccessToken(tokenString string, publicKey *rsa.PublicKey) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("algoritma signing tidak valid: %v", t.Header["alg"])
		}
		return publicKey, nil
	}, jwt.WithoutClaimsValidation()) // Allow expired tokens (untuk logout)

	if err != nil && !isTokenExpiredError(err) {
		return nil, fmt.Errorf("token JWT tidak valid: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("claims JWT tidak valid")
	}

	return claims, nil
}

// isTokenExpiredError memeriksa apakah error adalah expired token (masih boleh diproses untuk logout).
func isTokenExpiredError(err error) bool {
	return strings.Contains(err.Error(), "token is expired")
}
