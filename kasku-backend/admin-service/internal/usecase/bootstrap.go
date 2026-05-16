package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// BootstrapInput dipakai untuk menyemai admin pertama saat startup.
type BootstrapInput struct {
	Username string
	Password string
	Argon2   Argon2Config
}

// SeedBootstrapAdmin memastikan minimal ada satu SUPER_ADMIN di tabel admin_users.
// Idempotent: kalau sudah ada admin (Count > 0), langsung return tanpa side effect.
// Dipanggil dari main.go setelah migration sukses.
func SeedBootstrapAdmin(
	ctx context.Context,
	repo repository.AdminUserRepository,
	in BootstrapInput,
	log zerolog.Logger,
) error {
	count, err := repo.Count(ctx)
	if err != nil {
		return fmt.Errorf("gagal cek admin count: %w", err)
	}
	if count > 0 {
		log.Info().Int64("admin_count", count).Msg("bootstrap admin di-skip — sudah ada admin")
		return nil
	}

	if in.Username == "" || in.Password == "" {
		return fmt.Errorf("admin tabel kosong tapi ADMIN_BOOTSTRAP_USERNAME/PASSWORD tidak di-set")
	}

	hash, err := HashPassword(in.Password, in.Argon2)
	if err != nil {
		return fmt.Errorf("gagal hash bootstrap password: %w", err)
	}

	admin := &entity.AdminUser{
		ID:           uuid.New(),
		Username:     in.Username,
		PasswordHash: hash,
		Role:         entity.AdminRoleSuperAdmin,
		IsActive:     true,
	}
	if err := repo.CreateBootstrap(ctx, admin); err != nil {
		return fmt.Errorf("gagal insert bootstrap admin: %w", err)
	}

	log.Info().Str("username", in.Username).Msg("bootstrap admin di-seed")
	return nil
}
