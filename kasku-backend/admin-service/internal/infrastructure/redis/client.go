package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// adminJTIKeyPrefix memisahkan blacklist admin dari user JWT blacklist
	// (yang dipakai api-gateway dengan prefix "jti:").
	adminJTIKeyPrefix = "admin_jti:"
)

// TokenBlacklist mengelola JWT blacklist admin pakai Redis SET + TTL.
type TokenBlacklist struct {
	client *redis.Client
}

// NewTokenBlacklist membuat instance TokenBlacklist.
func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

// Blacklist menandai JTI sebagai revoked sampai TTL habis.
// Jika TTL <= 0, no-op (token sudah kedaluwarsa, JWT verify akan menolaknya).
func (tb *TokenBlacklist) Blacklist(ctx context.Context, jti string, remaining time.Duration) error {
	if remaining <= 0 {
		return nil
	}
	key := adminJTIKeyPrefix + jti
	if err := tb.client.Set(ctx, key, 1, remaining).Err(); err != nil {
		return fmt.Errorf("gagal blacklist admin JTI: %w", err)
	}
	return nil
}

// IsBlacklisted mengembalikan true bila JTI ada di blacklist.
func (tb *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := adminJTIKeyPrefix + jti
	count, err := tb.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("gagal cek admin JTI blacklist: %w", err)
	}
	return count > 0, nil
}

// NewClient membangun client Redis baru.
func NewClient(addr, password string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
}

// Ping memeriksa konektivitas Redis untuk health check.
func Ping(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}
