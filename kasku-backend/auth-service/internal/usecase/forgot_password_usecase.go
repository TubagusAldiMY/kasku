package usecase

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
)

const passwordResetTokenTTL = 1 * time.Hour

// ForgotPasswordUseCase mengimplementasikan alur lupa password.
type ForgotPasswordUseCase struct {
	userRepo  repository.UserRepository
	resetRepo repository.PasswordResetRepository
	publisher messaging.EventPublisher
}

// NewForgotPasswordUseCase membuat instance ForgotPasswordUseCase.
func NewForgotPasswordUseCase(
	userRepo repository.UserRepository,
	resetRepo repository.PasswordResetRepository,
	publisher messaging.EventPublisher,
) *ForgotPasswordUseCase {
	return &ForgotPasswordUseCase{
		userRepo:  userRepo,
		resetRepo: resetRepo,
		publisher: publisher,
	}
}

// Execute selalu return nil (anti-enumeration attack).
// Jika user ditemukan dan aktif: generate token reset, simpan di DB, publish event.
func (uc *ForgotPasswordUseCase) Execute(ctx context.Context, email string) error {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil || user == nil || !user.IsActive {
		// Sembunyikan detail — response selalu generik
		return nil
	}

	rawToken, tokenHash, err := generateSecureTokenWithHash()
	if err != nil {
		return nil
	}

	now := time.Now().UTC()
	resetToken := &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(passwordResetTokenTTL),
		CreatedAt: now,
	}

	if err := uc.resetRepo.Create(ctx, resetToken); err != nil {
		return nil
	}

	event := messaging.PasswordResetRequestedEvent{
		UserID:     user.ID.String(),
		Email:      user.Email,
		ResetToken: rawToken,
	}
	_ = uc.publisher.PublishPasswordResetRequested(ctx, event)

	return nil
}
