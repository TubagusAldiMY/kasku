package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// loginAttemptPrefix memisahkan counter brute-force login dari namespace lain.
	loginAttemptPrefix = "admin_login_fail:"
)

// LoginRateLimiter membatasi brute-force login admin dengan counter per-key di Redis.
// Counter naik pada setiap kegagalan (INCR + EXPIRE window) dan direset pada login sukses.
type LoginRateLimiter struct {
	client   *redis.Client
	maxFails int64
	window   time.Duration
}

// NewLoginRateLimiter membuat limiter. maxFails <= 0 atau window <= 0 dianggap config error.
func NewLoginRateLimiter(client *redis.Client, maxFails int64, window time.Duration) *LoginRateLimiter {
	if maxFails <= 0 {
		maxFails = 5
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	return &LoginRateLimiter{client: client, maxFails: maxFails, window: window}
}

// IsLocked mengembalikan true bila salah satu key (username atau IP) sudah melampaui
// batas kegagalan dalam window aktif. Dipanggil sebelum verifikasi password.
func (l *LoginRateLimiter) IsLocked(ctx context.Context, username, ip string) (bool, error) {
	for _, key := range l.keys(username, ip) {
		n, err := l.client.Get(ctx, key).Int64()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("gagal baca counter login: %w", err)
		}
		if n >= l.maxFails {
			return true, nil
		}
	}
	return false, nil
}

// RegisterFailure menaikkan counter untuk username DAN IP, menyetel TTL window pada
// increment pertama. Dipanggil setiap kali kredensial salah.
func (l *LoginRateLimiter) RegisterFailure(ctx context.Context, username, ip string) error {
	for _, key := range l.keys(username, ip) {
		n, err := l.client.Incr(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("gagal increment counter login: %w", err)
		}
		if n == 1 {
			if err := l.client.Expire(ctx, key, l.window).Err(); err != nil {
				return fmt.Errorf("gagal set TTL counter login: %w", err)
			}
		}
	}
	return nil
}

// Reset menghapus counter kedua key setelah login sukses.
func (l *LoginRateLimiter) Reset(ctx context.Context, username, ip string) error {
	if err := l.client.Del(ctx, l.keys(username, ip)...).Err(); err != nil {
		return fmt.Errorf("gagal reset counter login: %w", err)
	}
	return nil
}

// keys menghasilkan key per-username (case-insensitive) dan per-IP.
func (l *LoginRateLimiter) keys(username, ip string) []string {
	return []string{
		loginAttemptPrefix + "user:" + strings.ToLower(username),
		loginAttemptPrefix + "ip:" + ip,
	}
}
