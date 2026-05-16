package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/google/uuid"
)

// PasswordResetRepository mendefinisikan kontrak akses data untuk token reset password.
//
//go:generate mockgen -source=$GOFILE -destination=../../../tests/mocks/mock_password_reset_repository.go -package=mocks
type PasswordResetRepository interface {
	// Create menyimpan token reset password baru.
	Create(ctx context.Context, token *entity.PasswordResetToken) error

	// FindActiveByTokenHash mencari token reset yang aktif (belum digunakan, belum kadaluwarsa).
	// Mengembalikan nil, nil jika tidak ditemukan.
	FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.PasswordResetToken, error)

	// MarkAsUsed menandai token sebagai sudah digunakan.
	MarkAsUsed(ctx context.Context, tokenID uuid.UUID) error
}

// TransactionalResetPasswordRepository mendefinisikan operasi reset password dalam transaksi atomik.
type TransactionalResetPasswordRepository interface {
	// ExecuteResetPasswordTx menjalankan seluruh operasi reset password dalam satu transaksi:
	// 1. Update password_hash di users
	// 2. Mark token sebagai used
	// 3. Revoke semua refresh token aktif
	ExecuteResetPasswordTx(ctx context.Context, userID uuid.UUID, newPasswordHash string, tokenID uuid.UUID) error
}
