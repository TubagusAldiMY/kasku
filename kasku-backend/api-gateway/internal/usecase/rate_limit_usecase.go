package usecase

import (
	"context"
	"fmt"
	"time"
)

// RateLimitStore adalah interface untuk backend rate limiter (Redis).
type RateLimitStore interface {
	Check(ctx context.Context, key string, limit int64, window time.Duration) (*RateLimitCheckResult, error)
}

// RateLimitCheckResult berisi hasil pemeriksaan rate limit.
type RateLimitCheckResult struct {
	Allowed   bool
	Current   int64
	Limit     int64
	ResetTime time.Time
}

// RateLimitUseCase mengelola rate limiting per-endpoint dengan aturan yang berbeda.
type RateLimitUseCase struct {
	store RateLimitStore
}

// NewRateLimitUseCase membuat instance RateLimitUseCase baru.
func NewRateLimitUseCase(store RateLimitStore) *RateLimitUseCase {
	return &RateLimitUseCase{store: store}
}

// CheckRegister memeriksa rate limit untuk POST /v1/auth/register: 5 req/15min/IP.
func (uc *RateLimitUseCase) CheckRegister(ctx context.Context, clientIP string) (*RateLimitCheckResult, error) {
	key := fmt.Sprintf("register:ip:%s", clientIP)
	return uc.store.Check(ctx, key, 5, 15*time.Minute)
}

// CheckLogin memeriksa rate limit untuk POST /v1/auth/login:
//   - 10 req/min/IP
//   - 5 req/min/email (jika email tersedia)
func (uc *RateLimitUseCase) CheckLogin(ctx context.Context, clientIP, email string) (*RateLimitCheckResult, error) {
	// Cek limit per IP terlebih dahulu
	keyIP := fmt.Sprintf("login:ip:%s", clientIP)
	resultIP, err := uc.store.Check(ctx, keyIP, 10, time.Minute)
	if err != nil {
		return nil, err
	}
	if !resultIP.Allowed {
		return resultIP, nil
	}

	// Cek limit per email jika ada
	if email != "" {
		keyEmail := fmt.Sprintf("login:email:%s", email)
		resultEmail, err := uc.store.Check(ctx, keyEmail, 5, time.Minute)
		if err != nil {
			return nil, err
		}
		if !resultEmail.Allowed {
			return resultEmail, nil
		}
	}

	return resultIP, nil
}

// CheckRefresh memeriksa rate limit untuk POST /v1/auth/refresh: 20 req/min/user_id.
func (uc *RateLimitUseCase) CheckRefresh(ctx context.Context, userID string) (*RateLimitCheckResult, error) {
	key := fmt.Sprintf("refresh:user:%s", userID)
	return uc.store.Check(ctx, key, 20, time.Minute)
}

// CheckForgotPassword memeriksa rate limit untuk POST /v1/auth/forgot-password: 3 req/jam/email.
func (uc *RateLimitUseCase) CheckForgotPassword(ctx context.Context, email string) (*RateLimitCheckResult, error) {
	key := fmt.Sprintf("forgotpw:email:%s", email)
	return uc.store.Check(ctx, key, 3, time.Hour)
}

// CheckDefault memeriksa default rate limit: 200 req/min/user_id.
func (uc *RateLimitUseCase) CheckDefault(ctx context.Context, userID string) (*RateLimitCheckResult, error) {
	key := fmt.Sprintf("default:user:%s", userID)
	return uc.store.Check(ctx, key, 200, time.Minute)
}

// CheckSync memeriksa rate limit untuk /v1/sync/** : 60 req/min/user_id.
// Lebih ketat dari default karena setiap push bisa berisi batch operasi berat.
func (uc *RateLimitUseCase) CheckSync(ctx context.Context, userID string) (*RateLimitCheckResult, error) {
	key := fmt.Sprintf("sync:user:%s", userID)
	return uc.store.Check(ctx, key, 60, time.Minute)
}
