package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/messaging"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/ratelimit"
	"github.com/google/uuid"
)

const passwordResetTokenTTL = 1 * time.Hour

// EmailRateLimit menjelaskan rate-limit per-email untuk endpoint yang mengirim
// email (forgot-password, resend-verification). Limit & Window biasanya berasal
// dari configs.
type EmailRateLimit struct {
	Limit  int
	Window time.Duration
	// Endpoint adalah prefix Redis key (mis. "forgot:email", "resend:email").
	Endpoint string
}

// ForgotPasswordUseCase adalah kontrak alur lupa password.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_forgot_password_usecase.go -package=mocks
type ForgotPasswordUseCase interface {
	Execute(ctx context.Context, email string) error
}

// forgotPasswordUseCase mengimplementasikan ForgotPasswordUseCase.
type forgotPasswordUseCase struct {
	userRepo  repository.UserRepository
	resetRepo repository.PasswordResetRepository
	publisher messaging.EventPublisher
	limiter   ratelimit.Limiter
	limit     EmailRateLimit
}

// NewForgotPasswordUseCase membuat instance ForgotPasswordUseCase.
//
// Jika limiter == nil, rate-limit per-email dinonaktifkan (cocok untuk test/dev).
func NewForgotPasswordUseCase(
	userRepo repository.UserRepository,
	resetRepo repository.PasswordResetRepository,
	publisher messaging.EventPublisher,
	limiter ratelimit.Limiter,
	limit EmailRateLimit,
) ForgotPasswordUseCase {
	return &forgotPasswordUseCase{
		userRepo:  userRepo,
		resetRepo: resetRepo,
		publisher: publisher,
		limiter:   limiter,
		limit:     limit,
	}
}

// hashEmailForKey men-hash email (lowercase + trim) menjadi sha256 hex.
// Email tidak boleh tersimpan plaintext di Redis (PII) — hash satu arah cukup
// untuk identitas rate-limit.
func hashEmailForKey(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// Execute selalu return nil (anti-enumeration attack).
// Jika user ditemukan dan aktif: generate token reset, simpan di DB, publish event.
//
// Per-email rate limit dicek SEBELUM DB lookup untuk mencegah abuse:
//   - Mencegah penyerang spam email reset ke korban (mengganggu inbox + biaya kirim)
//   - Mencegah amplification attack pada SMTP/notification-service
//
// Saat rate-limit terlampaui, tetap return nil (silent) — tidak bocorkan apakah
// rate-limit aktif atau tidak.
func (uc *forgotPasswordUseCase) Execute(ctx context.Context, email string) error {
	if uc.limiter != nil && uc.limit.Limit > 0 && uc.limit.Window > 0 {
		key := fmt.Sprintf("ratelimit:%s:%s", uc.limit.Endpoint, hashEmailForKey(email))
		if _, err := uc.limiter.Check(ctx, key, uc.limit.Limit, uc.limit.Window); err != nil {
			if errors.Is(err, ratelimit.ErrLimitExceeded) {
				return nil
			}
			// Infra error → fail-open (lebih baik kirim email kadang-kadang daripada blocking semua user)
		}
	}

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
