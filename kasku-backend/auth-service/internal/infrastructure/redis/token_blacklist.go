package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	jtiKeyPrefix = "jti:"
)

// TokenBlacklist mengelola JWT blacklist menggunakan Redis SET dengan TTL.
// Setiap JTI yang di-blacklist disimpan dengan key "jti:{jti}" dan TTL sesuai sisa masa berlaku token.
type TokenBlacklist struct {
	client *redis.Client
}

// NewTokenBlacklist membuat instance TokenBlacklist.
func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

// BlacklistJTI menambahkan JTI ke blacklist dengan TTL = sisa masa berlaku token.
// Jika token sudah kadaluwarsa (remainingTTL <= 0), tidak perlu di-blacklist.
func (tb *TokenBlacklist) BlacklistJTI(ctx context.Context, jti string, remainingTTL time.Duration) error {
	if remainingTTL <= 0 {
		// Token sudah kadaluwarsa — tidak perlu di-blacklist karena verifikasi JWT akan menolaknya
		return nil
	}

	key := jtiKeyPrefix + jti
	if err := tb.client.Set(ctx, key, 1, remainingTTL).Err(); err != nil {
		return fmt.Errorf("gagal blacklist JTI ke Redis: %w", err)
	}

	return nil
}

// IsJTIBlacklisted memeriksa apakah JTI ada di blacklist.
func (tb *TokenBlacklist) IsJTIBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := jtiKeyPrefix + jti
	result, err := tb.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("gagal cek blacklist JTI dari Redis: %w", err)
	}
	return result > 0, nil
}

// NewRedisClient membuat Redis client dengan konfigurasi yang diberikan.
func NewRedisClient(addr, password string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
}

// PingRedis memeriksa koneksi ke Redis untuk health check.
func PingRedis(ctx context.Context, client *redis.Client) error {
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("gagal ping Redis: %w", err)
	}
	return nil
}
