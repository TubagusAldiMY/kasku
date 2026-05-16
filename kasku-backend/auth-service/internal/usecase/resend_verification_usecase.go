package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/messaging"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/ratelimit"
	"github.com/google/uuid"
)

// ResendVerificationUseCase adalah kontrak alur pengiriman ulang email verifikasi.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_resend_verification_usecase.go -package=mocks
type ResendVerificationUseCase interface {
	Execute(ctx context.Context, email string) error
}

// resendVerificationUseCase mengimplementasikan ResendVerificationUseCase.
type resendVerificationUseCase struct {
	userRepo       repository.UserRepository
	emailVerifRepo repository.EmailVerificationRepository
	publisher      messaging.EventPublisher
	limiter        ratelimit.Limiter
	limit          EmailRateLimit
}

// NewResendVerificationUseCase membuat instance ResendVerificationUseCase.
//
// Jika limiter == nil, rate-limit per-email dinonaktifkan (cocok untuk test/dev).
func NewResendVerificationUseCase(
	userRepo repository.UserRepository,
	emailVerifRepo repository.EmailVerificationRepository,
	publisher messaging.EventPublisher,
	limiter ratelimit.Limiter,
	limit EmailRateLimit,
) ResendVerificationUseCase {
	return &resendVerificationUseCase{
		userRepo:       userRepo,
		emailVerifRepo: emailVerifRepo,
		publisher:      publisher,
		limiter:        limiter,
		limit:          limit,
	}
}

// Execute menjalankan alur resend verifikasi:
// - Selalu return sukses untuk mencegah email enumeration attack
// - Jika user ditemukan dan belum terverifikasi: invalidate token lama, buat token baru, publish event
//
// Per-email rate limit dicek SEBELUM DB lookup untuk mencegah penyerang spam email
// resend ke korban. Saat rate-limit terlampaui, tetap return nil (silent).
func (uc *resendVerificationUseCase) Execute(ctx context.Context, email string) error {
	if uc.limiter != nil && uc.limit.Limit > 0 && uc.limit.Window > 0 {
		key := fmt.Sprintf("ratelimit:%s:%s", uc.limit.Endpoint, hashEmailForKey(email))
		if _, err := uc.limiter.Check(ctx, key, uc.limit.Limit, uc.limit.Window); err != nil {
			if errors.Is(err, ratelimit.ErrLimitExceeded) {
				return nil
			}
			// Infra error → fail-open
		}
	}

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
