package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/google/uuid"
)

// EmailVerificationRepository mendefinisikan kontrak akses data untuk token verifikasi email.
type EmailVerificationRepository interface {
	// Create menyimpan token verifikasi email baru.
	Create(ctx context.Context, verification *entity.EmailVerification) error

	// FindActiveByTokenHash mencari token verifikasi aktif (belum digunakan, belum kadaluwarsa).
	// Mengembalikan nil, nil jika tidak ditemukan.
	FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.EmailVerification, error)

	// MarkAsVerified menandai token sebagai sudah digunakan dengan mengisi verified_at.
	MarkAsVerified(ctx context.Context, verificationID uuid.UUID) error

	// InvalidateAllActiveByUserID menginvalidasi semua token aktif milik user.
	// Digunakan saat resend verification untuk mencegah token lama masih bisa dipakai.
	InvalidateAllActiveByUserID(ctx context.Context, userID uuid.UUID) error
}
