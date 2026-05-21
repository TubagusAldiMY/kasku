package usecase

import (
	"context"
	"fmt"

	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/google/uuid"
)

// ChangePasswordUseCase adalah kontrak alur ganti password dari profil.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_change_password_usecase.go -package=mocks
type ChangePasswordUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
}

type changePasswordUseCase struct {
	userRepo  repository.UserRepository
	argon2Cfg Argon2Config
}

func NewChangePasswordUseCase(userRepo repository.UserRepository, argon2Cfg Argon2Config) ChangePasswordUseCase {
	return &changePasswordUseCase{
		userRepo:  userRepo,
		argon2Cfg: argon2Cfg,
	}
}

// Execute menjalankan alur ganti password:
// 1. Fetch user by ID
// 2. Verifikasi password saat ini dengan Argon2id constant-time compare
// 3. Validasi kekuatan password baru
// 4. Hash password baru dengan Argon2id
// 5. Persist ke database
func (uc *changePasswordUseCase) Execute(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("gagal fetch user: %w", err)
	}
	if user == nil {
		return domainerrors.ErrUserNotFound
	}

	if !verifyArgon2idPassword(currentPassword, user.PasswordHash) {
		return domainerrors.ErrInvalidCredentials
	}

	if err := validatePassword(newPassword); err != nil {
		return err
	}

	newHash, err := hashPasswordArgon2id(newPassword, uc.argon2Cfg)
	if err != nil {
		return fmt.Errorf("gagal hash password baru: %w", err)
	}

	if err := uc.userRepo.UpdatePassword(ctx, userID, newHash); err != nil {
		return fmt.Errorf("gagal update password: %w", err)
	}

	return nil
}