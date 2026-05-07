package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
)

// ResendVerificationUseCase mengimplementasikan alur pengiriman ulang email verifikasi.
type ResendVerificationUseCase struct {
	userRepo       repository.UserRepository
	emailVerifRepo repository.EmailVerificationRepository
	publisher      messaging.EventPublisher
}

// NewResendVerificationUseCase membuat instance ResendVerificationUseCase.
func NewResendVerificationUseCase(
	userRepo repository.UserRepository,
	emailVerifRepo repository.EmailVerificationRepository,
	publisher messaging.EventPublisher,
) *ResendVerificationUseCase {
	return &ResendVerificationUseCase{
		userRepo:       userRepo,
		emailVerifRepo: emailVerifRepo,
		publisher:      publisher,
	}
}

// Execute menjalankan alur resend verifikasi:
// - Selalu return sukses untuk mencegah email enumeration attack
// - Jika user ditemukan dan belum terverifikasi: invalidate token lama, buat token baru, publish event
func (uc *ResendVerificationUseCase) Execute(ctx context.Context, email string) error {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("gagal lookup user: %w", err)
	}

	// Tidak reveal apakah email ada atau tidak (anti-enumeration)
	if user == nil || user.EmailVerified {
		// Tetap return nil — response selalu generik
		return nil
	}

	// Invalidate semua token verifikasi aktif lama
	if err := uc.emailVerifRepo.InvalidateAllActiveByUserID(ctx, user.ID); err != nil {
		return fmt.Errorf("gagal invalidate token lama: %w", err)
	}

	rawToken, tokenHash, err := generateSecureTokenWithHash()
	if err != nil {
		return fmt.Errorf("gagal generate token baru: %w", err)
	}

	now := time.Now().UTC()
	newVerification := &entity.EmailVerification{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(emailVerificationTokenTTL),
		CreatedAt: now,
	}

	if err := uc.emailVerifRepo.Create(ctx, newVerification); err != nil {
		return fmt.Errorf("gagal simpan token verifikasi baru: %w", err)
	}

	event := messaging.EmailVerificationResentEvent{
		UserID:            user.ID.String(),
		Email:             user.Email,
		VerificationToken: rawToken,
	}

	// Publish gagal → log di handler, tapi use case tetap return nil (silent fail untuk anti-enumeration)
	_ = uc.publisher.PublishEmailVerificationResent(ctx, event)

	return nil
}
