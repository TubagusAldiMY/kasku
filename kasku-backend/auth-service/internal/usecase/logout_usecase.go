package usecase

import (
	"context"
	"crypto/rsa"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
)

// TokenBlacklister mendefinisikan kontrak untuk blacklist JWT.
type TokenBlacklister interface {
	BlacklistJTI(ctx context.Context, jti string, remainingTTL time.Duration) error
}

// LogoutInput berisi data yang diperlukan untuk proses logout.
type LogoutInput struct {
	AccessToken     string
	RawRefreshToken string
}

// LogoutUseCase mengimplementasikan alur logout user.
type LogoutUseCase struct {
	refreshTokenRepo repository.RefreshTokenRepository
	jwtPublicKey     *rsa.PublicKey
	blacklist        TokenBlacklister
}

// NewLogoutUseCase membuat instance LogoutUseCase.
func NewLogoutUseCase(
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtPublicKey *rsa.PublicKey,
	blacklist TokenBlacklister,
) *LogoutUseCase {
	return &LogoutUseCase{
		refreshTokenRepo: refreshTokenRepo,
		jwtPublicKey:     jwtPublicKey,
		blacklist:        blacklist,
	}
}

// Execute menjalankan alur logout:
// 1. Blacklist JTI access token di Redis (agar token tidak bisa dipakai lagi meski belum expired)
// 2. Revoke refresh token di DB
// Selalu return nil — error saat logout tidak boleh gagalkan response.
func (uc *LogoutUseCase) Execute(ctx context.Context, input LogoutInput) error {
	// Blacklist access token JTI jika tersedia
	if input.AccessToken != "" {
		claims, err := ParseAccessToken(input.AccessToken, uc.jwtPublicKey)
		if err == nil && claims != nil && claims.ID != "" {
			remaining := time.Until(claims.ExpiresAt.Time)
			_ = uc.blacklist.BlacklistJTI(ctx, claims.ID, remaining)
		}
	}

	// Revoke refresh token jika tersedia
	if input.RawRefreshToken != "" {
		tokenHash := hashToken(input.RawRefreshToken)
		token, err := uc.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
		if err == nil && token != nil && !token.IsRevoked {
			_ = uc.refreshTokenRepo.RevokeByID(ctx, token.ID)
		}
	}

	return nil
}
