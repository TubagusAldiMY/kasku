package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/TubagusAldiMY/kasku/api-gateway/internal/usecase"
)

// RateLimiter mengimplementasikan usecase.RateLimitStore dengan Redis backend.
// Algoritma: INCR key + TTL window (atomic via pipeline).
// Key format: ratelimit:{scope}:{identifier}
type RateLimiter struct {
	client *goredis.Client
}

// NewRateLimiter membuat instance RateLimiter baru.
func NewRateLimiter(client *goredis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Check memeriksa apakah request masih dalam batas rate limit.
// Mengimplementasikan usecase.RateLimitStore.
func (r *RateLimiter) Check(ctx context.Context, key string, limit int64, window time.Duration) (*usecase.RateLimitCheckResult, error) {
	now := time.Now().UTC()
	windowKey := fmt.Sprintf("ratelimit:%s", key)

	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("gagal eksekusi rate limit pipeline: %w", err)
	}

	current := incrCmd.Val()
	ttl, err := r.client.TTL(ctx, windowKey).Result()
	if err != nil || ttl < 0 {
		ttl = window
	}

	return &usecase.RateLimitCheckResult{
		Allowed:   current <= limit,
		Current:   current,
		Limit:     limit,
		ResetTime: now.Add(ttl),
	}, nil
}

// NewRedisClient membuat instance Redis client baru.
func NewRedisClient(addr, password string) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
}

// Ping memeriksa koneksi Redis.
func Ping(ctx context.Context, client *goredis.Client) error {
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("gagal ping Redis: %w", err)
	}
	return nil
}
