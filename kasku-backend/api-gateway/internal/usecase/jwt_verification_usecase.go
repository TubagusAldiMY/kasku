package usecase

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// KasKuClaims mendefinisikan JWT claims yang diterbitkan oleh auth-service.
type KasKuClaims struct {
	jwt.RegisteredClaims
	Email            string `json:"email"`
	TenantSchema     string `json:"tenant_schema"`
	SubscriptionTier string `json:"subscription_tier"`
}

// ParsedToken berisi data yang sudah diverifikasi dari JWT.
type ParsedToken struct {
	UserID           uuid.UUID
	JTI              string
	Email            string
	TenantSchema     string
	SubscriptionTier string
	ExpiresAt        time.Time
}

// BlacklistChecker adalah interface untuk memeriksa apakah JTI ada di blacklist.
type BlacklistChecker interface {
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// JWTVerificationUseCase memverifikasi JWT RS256 dan memeriksa blacklist.
type JWTVerificationUseCase struct {
	publicKey *rsa.PublicKey
	blacklist BlacklistChecker
}

// NewJWTVerificationUseCase membuat instance JWTVerificationUseCase baru.
func NewJWTVerificationUseCase(publicKey *rsa.PublicKey, blacklist BlacklistChecker) *JWTVerificationUseCase {
	return &JWTVerificationUseCase{
		publicKey: publicKey,
		blacklist: blacklist,
	}
}

// Verify memverifikasi token JWT dan mengembalikan ParsedToken jika valid.
// Error dikembalikan jika:
//   - Token tidak valid / expired
//   - Algorithm bukan RS256 (algorithm confusion protection)
//   - JTI ada di blacklist Redis
func (uc *JWTVerificationUseCase) Verify(ctx context.Context, tokenString string) (*ParsedToken, error) {
	// Parse dengan validasi ketat: hanya terima RS256
	token, err := jwt.ParseWithClaims(
		tokenString,
		&KasKuClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Algorithm confusion protection: tolak jika bukan RS256
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, fmt.Errorf("algorithm %s tidak diizinkan, hanya RS256", token.Method.Alg())
			}
			return uc.publicKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("token tidak valid: %w", err)
	}

	claims, ok := token.Claims.(*KasKuClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token claims tidak valid")
	}

	// Validasi JTI wajib ada
	jti := claims.ID
	if jti == "" {
		return nil, fmt.Errorf("token tidak memiliki JTI")
	}

	// Cek blacklist di Redis
	blacklisted, err := uc.blacklist.IsBlacklisted(ctx, jti)
	if err != nil {
		// Jika Redis error, tolak request (fail-secure)
		return nil, fmt.Errorf("tidak dapat memverifikasi status token: %w", err)
	}
	if blacklisted {
		return nil, fmt.Errorf("token sudah direvoke")
	}

	// Parse user_id dari subject
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("subject token bukan UUID yang valid: %w", err)
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return &ParsedToken{
		UserID:           userID,
		JTI:              jti,
		Email:            claims.Email,
		TenantSchema:     claims.TenantSchema,
		SubscriptionTier: claims.SubscriptionTier,
		ExpiresAt:        expiresAt,
	}, nil
}
