package usecase

import (
	"context"
	"fmt"

	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
)

// VerifyEmailUseCase adalah kontrak alur verifikasi email.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_verify_email_usecase.go -package=mocks
type VerifyEmailUseCase interface {
	Execute(ctx context.Context, rawToken string) error
}

// verifyEmailUseCase mengimplementasikan VerifyEmailUseCase.
type verifyEmailUseCase struct {
	emailVerifRepo repository.EmailVerificationRepository
	userRepo       repository.UserRepository
}

// NewVerifyEmailUseCase membuat instance VerifyEmailUseCase.
func NewVerifyEmailUseCase(
	emailVerifRepo repository.EmailVerificationRepository,
	userRepo repository.UserRepository,
) VerifyEmailUseCase {
	return &verifyEmailUseCase{
		emailVerifRepo: emailVerifRepo,
		userRepo:       userRepo,
	}
}

// Execute menjalankan alur verifikasi email:
// 1. SHA256(token) → lookup email_verifications aktif
// 2. Tandai token sebagai verified
// 3. Update user: is_active=true, email_verified=true
func (uc *verifyEmailUseCase) Execute(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return fmt.Errorf("%w: token tidak boleh kosong", domainerrors.ErrInvalidToken)
	}

	tokenHash := hashToken(rawToken)

	verification, err := uc.emailVerifRepo.FindActiveByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("gagal lookup verification token: %w", err)
	}
	if verification == nil {
		return domainerrors.ErrInvalidToken
	}

	if err := uc.emailVerifRepo.MarkAsVerified(ctx, verification.ID); err != nil {
		return fmt.Errorf("gagal mark verification: %w", err)
	}

	if err := uc.userRepo.VerifyEmail(ctx, verification.UserID); err != nil {
		return fmt.Errorf("gagal update user verified: %w", err)
	}

	return nil
}
