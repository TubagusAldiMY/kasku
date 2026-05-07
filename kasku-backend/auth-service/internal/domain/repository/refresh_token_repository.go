package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/google/uuid"
)

// RefreshTokenRepository mendefinisikan kontrak akses data untuk entitas RefreshToken.
type RefreshTokenRepository interface {
	// Create menyimpan refresh token baru.
	Create(ctx context.Context, token *entity.RefreshToken) error

	// FindByTokenHash mencari refresh token berdasarkan hash-nya.
	// Mengembalikan nil, nil jika tidak ditemukan.
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)

	// RevokeByID mencabut satu refresh token berdasarkan ID-nya.
	RevokeByID(ctx context.Context, tokenID uuid.UUID) error

	// RevokeAllActiveByUserID mencabut semua refresh token aktif milik user.
	// Digunakan saat token reuse attack terdeteksi.
	RevokeAllActiveByUserID(ctx context.Context, userID uuid.UUID) error

	// RevokeAllByUserIDInTx mencabut semua refresh token aktif dalam transaksi yang ada.
	// Digunakan saat reset password untuk memastikan atomisitas.
	RevokeAllByUserIDInTx(ctx context.Context, tx interface{}, userID uuid.UUID) error
}
