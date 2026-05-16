package usecase

import (
	"context"
	"fmt"

	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
)

// ResetPasswordUseCase adalah kontrak alur reset password.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_reset_password_usecase.go -package=mocks
type ResetPasswordUseCase interface {
	Execute(ctx context.Context, rawToken, newPassword string) error
}

// resetPasswordUseCase mengimplementasikan ResetPasswordUseCase.
type resetPasswordUseCase struct {
	resetRepo   repository.PasswordResetRepository
	resetTxRepo repository.TransactionalResetPasswordRepository
	argon2Cfg   Argon2Config
}

// NewResetPasswordUseCase membuat instance ResetPasswordUseCase.
func NewResetPasswordUseCase(
	resetRepo repository.PasswordResetRepository,
	resetTxRepo repository.TransactionalResetPasswordRepository,
	argon2Cfg Argon2Config,
) ResetPasswordUseCase {
	return &resetPasswordUseCase{
		resetRepo:   resetRepo,
		resetTxRepo: resetTxRepo,
		argon2Cfg:   argon2Cfg,
	}
}

// Execute menjalankan alur reset password:
// 1. Validasi password baru
// 2. SHA256(token) → lookup token aktif
// 3. Hash password baru dengan Argon2id
// 4. Atomic DB transaction: update password + mark token used + revoke semua refresh tokens
func (uc *resetPasswordUseCase) Execute(ctx context.Context, rawToken, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	tokenHash := hashToken(rawToken)

	token, err := uc.resetRepo.FindActiveByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("gagal lookup reset token: %w", err)
	}
	if token == nil {
		return domainerrors.ErrInvalidToken
	}

	newHash, err := hashPasswordArgon2id(newPassword, uc.argon2Cfg)
	if err != nil {
		return fmt.Errorf("gagal hash password baru: %w", err)
	}

	return uc.resetTxRepo.ExecuteResetPasswordTx(ctx, token.UserID, newHash, token.ID)
}
