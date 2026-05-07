package usecase

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/google/uuid"
)

// RefreshTokenUseCase mengimplementasikan alur refresh access token dengan rotasi.
// Mendeteksi token reuse attack menggunakan "refresh token rotation" pattern.
type RefreshTokenUseCase struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtPrivateKey    *rsa.PrivateKey
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

// NewRefreshTokenUseCase membuat instance RefreshTokenUseCase.
func NewRefreshTokenUseCase(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtPrivateKey *rsa.PrivateKey,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtPrivateKey:    jwtPrivateKey,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

// RefreshInput berisi data yang diperlukan untuk refresh token.
type RefreshInput struct {
	RawRefreshToken string
	UserAgent       string
	IPAddress       string
	IsDev           bool
}

// Execute menjalankan alur refresh token dengan deteksi token reuse:
// 1. SHA256(cookie) → lookup refresh token
// 2. Jika tidak ditemukan → 401 INVALID_TOKEN
// 3. Jika is_revoked=true → TOKEN_REUSE_DETECTED → revoke semua token user
// 4. Jika expired → 401 TOKEN_EXPIRED
// 5. Revoke token lama, issue token baru (rotasi)
func (uc *RefreshTokenUseCase) Execute(ctx context.Context, input RefreshInput) (*LoginOutput, error) {
	if input.RawRefreshToken == "" {
		return nil, domainerrors.ErrInvalidToken
	}

	tokenHash := hashToken(input.RawRefreshToken)

	existingToken, err := uc.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("gagal lookup refresh token: %w", err)
	}
	if existingToken == nil {
		return nil, domainerrors.ErrInvalidToken
	}

	// REUSE DETECTION: token sudah di-revoke → kemungkinan token theft
	if existingToken.IsRevoked {
		// Revoke SEMUA token aktif milik user untuk memaksa login ulang di semua device
		_ = uc.refreshTokenRepo.RevokeAllActiveByUserID(ctx, existingToken.UserID)
		return nil, domainerrors.ErrTokenReuseDetected
	}

	now := time.Now().UTC()
	if existingToken.IsExpired(now) {
		return nil, domainerrors.ErrInvalidToken
	}

	// Revoke token lama sebelum issue yang baru (refresh token rotation)
	if err := uc.refreshTokenRepo.RevokeByID(ctx, existingToken.ID); err != nil {
		return nil, fmt.Errorf("gagal revoke token lama: %w", err)
	}

	user, err := uc.userRepo.FindByID(ctx, existingToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("gagal lookup user: %w", err)
	}
	if user == nil {
		return nil, domainerrors.ErrInvalidToken
	}

	// Issue access token baru
	accessToken, err := GenerateAccessToken(user, uc.jwtPrivateKey, uc.accessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("gagal generate access token baru: %w", err)
	}

	// Issue refresh token baru
	rawToken, newTokenHash, err := generateSecureTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("gagal generate refresh token baru: %w", err)
	}

	userAgent := &input.UserAgent
	ipAddress := &input.IPAddress
	if input.UserAgent == "" {
		userAgent = nil
	}
	if input.IPAddress == "" {
		ipAddress = nil
	}

	newRefreshToken := &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: newTokenHash,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: now.Add(uc.refreshTokenTTL),
		CreatedAt: now,
	}

	if err := uc.refreshTokenRepo.Create(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("gagal simpan refresh token baru: %w", err)
	}

	return &LoginOutput{
		AccessToken: accessToken,
		TokenType:   tokenTypeBearear,
		ExpiresIn:   int64(uc.accessTokenTTL.Seconds()),
		RefreshTokenCookie: RefreshTokenCookieParams{
			RawToken: rawToken,
			MaxAge:   int(uc.refreshTokenTTL.Seconds()),
			IsSecure: !input.IsDev,
		},
	}, nil
}
