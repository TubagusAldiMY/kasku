package usecase

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"

	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/golang-jwt/jwt/v5"
)

// TokenBlacklistChecker memeriksa apakah JTI sudah dicabut.
// Dipisah dari implementasi Redis konkret agar use case bisa di-test dengan mock.
type TokenBlacklistChecker interface {
	IsJTIBlacklisted(ctx context.Context, jti string) (bool, error)
}

// ValidateAccessTokenUseCase adalah kontrak verifikasi STRICT atas access token:
//   - Signature RS256 valid
//   - Token belum expired
//   - JTI tidak ada di blacklist
//
// Berbeda dengan jwt_helpers.ParseAccessToken yang sengaja mengizinkan token expired
// (untuk endpoint /logout supaya JTI tetap bisa di-blacklist).
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_validate_token_usecase.go -package=mocks
type ValidateAccessTokenUseCase interface {
	Execute(ctx context.Context, tokenString string) (*JWTClaims, error)
}

// validateAccessTokenUseCase mengimplementasikan ValidateAccessTokenUseCase.
type validateAccessTokenUseCase struct {
	publicKey *rsa.PublicKey
	blacklist TokenBlacklistChecker
}

// NewValidateAccessTokenUseCase membuat instance ValidateAccessTokenUseCase.
func NewValidateAccessTokenUseCase(publicKey *rsa.PublicKey, blacklist TokenBlacklistChecker) ValidateAccessTokenUseCase {
	return &validateAccessTokenUseCase{
		publicKey: publicKey,
		blacklist: blacklist,
	}
}

// Execute melakukan verifikasi strict. Mengembalikan ErrInvalidToken untuk
// semua kegagalan validasi (signature, expired, blacklisted) — jangan bocorkan
// detail spesifik ke client.
func (uc *validateAccessTokenUseCase) Execute(ctx context.Context, tokenString string) (*JWTClaims, error) {
	if tokenString == "" {
		return nil, domainerrors.ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("algoritma signing tidak valid: %v", t.Header["alg"])
		}
		return uc.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domainerrors.ErrInvalidToken
		}
		return nil, domainerrors.ErrInvalidToken
	}

	if !token.Valid {
		return nil, domainerrors.ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, domainerrors.ErrInvalidToken
	}

	blacklisted, err := uc.blacklist.IsJTIBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("gagal cek blacklist: %w", err)
	}
	if blacklisted {
		return nil, domainerrors.ErrInvalidToken
	}

	return claims, nil
}
