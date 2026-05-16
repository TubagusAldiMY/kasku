// Package ratelimit menyediakan implementasi rate limiter berbasis Redis
// sliding-window log. Dipakai oleh HTTP middleware (rate limit per-IP) dan
// use case (rate limit per-email pada /forgot-password & /resend-verification).
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrLimitExceeded dikembalikan saat request melebihi limit dalam window.
// Caller bisa memeriksa dengan errors.Is(err, ErrLimitExceeded).
var ErrLimitExceeded = errors.New("rate limit exceeded")

// slidingWindowScript adalah Lua script atomic untuk sliding-window rate limit
// menggunakan Redis sorted set. Mengembalikan:
//
//	{1, 0}            → request diterima
//	{0, oldestScore}  → request ditolak, oldestScore adalah timestamp Unix detik
//	                    dari entry tertua di window (untuk hitung Retry-After).
//
// Argumen:
//
//	KEYS[1] = key
//	ARGV[1] = nowUnix
//	ARGV[2] = windowSeconds
//	ARGV[3] = limit
//	ARGV[4] = unique member (ZSet butuh member unik agar entry tidak overwrite)
var slidingWindowScript = redis.NewScript(`
local key       = KEYS[1]
local now       = tonumber(ARGV[1])
local window    = tonumber(ARGV[2])
local limit     = tonumber(ARGV[3])
local member    = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
  local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
  if #oldest >= 2 then
    return {0, tonumber(oldest[2])}
  end
  return {0, now}
end
redis.call('ZADD', key, now, member)
redis.call('EXPIRE', key, window)
return {1, 0}
`)

// Limiter adalah kontrak untuk rate limiter.
// Implementasi default adalah RedisLimiter.
//
//go:generate mockgen -source=$GOFILE -destination=../../../tests/mocks/mock_limiter.go -package=mocks
type Limiter interface {
	// Check memeriksa apakah request boleh lewat. Mengembalikan:
	//   - retryAfter: berapa lama harus menunggu sebelum bisa retry (0 saat allowed)
	//   - err: nil saat allowed, ErrLimitExceeded saat ditolak, error lain saat infra gagal
	Check(ctx context.Context, key string, limit int, window time.Duration) (retryAfter time.Duration, err error)
}

// RedisLimiter mengimplementasikan Limiter menggunakan Redis sorted set.
type RedisLimiter struct {
	rdb *redis.Client
}

// NewRedisLimiter membuat RedisLimiter baru.
func NewRedisLimiter(rdb *redis.Client) *RedisLimiter {
	return &RedisLimiter{rdb: rdb}
}

// Check mengeksekusi sliding window Lua script.
func (l *RedisLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (time.Duration, error) {
	if limit <= 0 || window <= 0 {
		return 0, fmt.Errorf("ratelimit: limit dan window harus > 0")
	}

	now := time.Now().Unix()
	member := fmt.Sprintf("%d:%d", now, time.Now().UnixNano())
	windowSec := int(window.Seconds())

	res, err := slidingWindowScript.Run(ctx, l.rdb, []string{key},
		now, windowSec, limit, member,
	).Result()
	if err != nil {
		return 0, fmt.Errorf("ratelimit redis: %w", err)
	}

	arr, ok := res.([]any)
	if !ok || len(arr) < 2 {
		return 0, fmt.Errorf("ratelimit: response Lua tidak terduga")
	}

	allowed, _ := arr[0].(int64)
	if allowed == 1 {
		return 0, nil
	}

	oldestUnix, _ := arr[1].(int64)
	retryAfter := max(
		time.Duration(int64(windowSec)-(now-oldestUnix))*time.Second,
		time.Second,
	)
	return retryAfter, ErrLimitExceeded
}
