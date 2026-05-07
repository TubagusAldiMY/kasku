package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/google/uuid"
)

// UserRepository mendefinisikan kontrak akses data untuk entitas User.
// Implementasi konkret berada di infrastructure/persistence.
type UserRepository interface {
	// FindByEmail mencari user berdasarkan email (case-insensitive).
	// Mengembalikan nil, nil jika tidak ditemukan.
	FindByEmail(ctx context.Context, email string) (*entity.User, error)

	// FindByID mencari user berdasarkan UUID.
	// Mengembalikan nil, nil jika tidak ditemukan.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)

	// ExistsByEmail memeriksa apakah email sudah terdaftar (case-insensitive).
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByUsername memeriksa apakah username sudah digunakan (case-insensitive).
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// Create menyimpan user baru dalam database.
	Create(ctx context.Context, user *entity.User) error

	// UpdateLoginSuccess mereset failed_login_count dan meng-update last_login_at.
	UpdateLoginSuccess(ctx context.Context, userID uuid.UUID) error

	// IncrementFailedLoginCount menaikkan failed_login_count.
	// Jika count mencapai maxAttempts, set locked_until.
	IncrementFailedLoginAndLockIfNeeded(ctx context.Context, userID uuid.UUID, maxAttempts int16, lockoutDuration string) error

	// VerifyEmail meng-update is_active=true dan email_verified=true.
	VerifyEmail(ctx context.Context, userID uuid.UUID) error

	// UpdatePassword meng-update password_hash user.
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error
}
