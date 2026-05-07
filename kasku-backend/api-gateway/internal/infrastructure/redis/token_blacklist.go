package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// TokenBlacklist memeriksa apakah sebuah JTI ada di blacklist Redis.
// Key format: jti:{jti} — di-set oleh auth-service saat logout.
type TokenBlacklist struct {
	client *goredis.Client
}

// NewTokenBlacklist membuat instance TokenBlacklist baru.
func NewTokenBlacklist(client *goredis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

// IsBlacklisted mengembalikan true jika jti ada di blacklist Redis.
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("jti:%s", jti)
	result, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("gagal cek blacklist JTI di Redis: %w", err)
	}
	return result > 0, nil
}
