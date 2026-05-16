package ratelimit_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/ratelimit"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRedis spawns miniredis (in-memory) — tidak butuh Docker.
func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client, mr
}

func TestRedisLimiter_Check(t *testing.T) {
	t.Parallel()

	t.Run("invalid args: limit=0 or window=0 → error", func(t *testing.T) {
		t.Parallel()
		client, _ := newTestRedis(t)
		lim := ratelimit.NewRedisLimiter(client)

		_, err := lim.Check(context.Background(), "k", 0, time.Minute)
		assert.Error(t, err)

		_, err = lim.Check(context.Background(), "k", 10, 0)
		assert.Error(t, err)
	})

	t.Run("under limit: allow all + retryAfter=0", func(t *testing.T) {
		t.Parallel()
		client, _ := newTestRedis(t)
		lim := ratelimit.NewRedisLimiter(client)

		for i := range 5 {
			ra, err := lim.Check(context.Background(), "user:abc", 10, time.Minute)
			require.NoError(t, err, "iteration %d", i)
			assert.Equal(t, time.Duration(0), ra)
		}
	})

	t.Run("over limit: ErrLimitExceeded + positive retryAfter", func(t *testing.T) {
		t.Parallel()
		client, _ := newTestRedis(t)
		lim := ratelimit.NewRedisLimiter(client)

		const limit = 3
		for i := range limit {
			ra, err := lim.Check(context.Background(), "u:over", limit, time.Minute)
			require.NoError(t, err, "iteration %d", i)
			assert.Equal(t, time.Duration(0), ra)
		}

		ra, err := lim.Check(context.Background(), "u:over", limit, time.Minute)
		assert.ErrorIs(t, err, ratelimit.ErrLimitExceeded)
		assert.Greater(t, ra, time.Duration(0))
	})

	t.Run("key isolation: 2 key tidak saling block", func(t *testing.T) {
		t.Parallel()
		client, _ := newTestRedis(t)
		lim := ratelimit.NewRedisLimiter(client)

		for range 3 {
			_, err := lim.Check(context.Background(), "a", 3, time.Minute)
			require.NoError(t, err)
		}
		// key "a" sudah habis quota, key "b" masih fresh
		ra, err := lim.Check(context.Background(), "b", 3, time.Minute)
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), ra)
	})

	t.Run("TTL expiry: setelah window berlalu, counter reset", func(t *testing.T) {
		t.Parallel()
		client, mr := newTestRedis(t)
		lim := ratelimit.NewRedisLimiter(client)

		const limit = 2
		for range limit {
			_, err := lim.Check(context.Background(), "exp:k", limit, 10*time.Second)
			require.NoError(t, err)
		}
		// over limit
		_, err := lim.Check(context.Background(), "exp:k", limit, 10*time.Second)
		assert.ErrorIs(t, err, ratelimit.ErrLimitExceeded)

		// fast-forward miniredis time melewati window
		mr.FastForward(15 * time.Second)

		// counter cleaned by ZREMRANGEBYSCORE — should allow again
		_, err = lim.Check(context.Background(), "exp:k", limit, 10*time.Second)
		assert.NoError(t, err)
	})
}

// TestRedisLimiter_Concurrent verifies the Lua script atomicity:
// dengan N goroutine masing-masing memanggil Check, exactly `limit` saja
// yang boleh lewat (sisanya dapat ErrLimitExceeded). Tidak boleh race.
func TestRedisLimiter_Concurrent(t *testing.T) {
	t.Parallel()
	client, _ := newTestRedis(t)
	lim := ratelimit.NewRedisLimiter(client)

	const (
		limit       = 10
		concurrency = 50
	)

	var allowed, denied int64
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			_, err := lim.Check(context.Background(), "concur:key", limit, time.Minute)
			if err == nil {
				atomic.AddInt64(&allowed, 1)
				return
			}
			if errors.Is(err, ratelimit.ErrLimitExceeded) {
				atomic.AddInt64(&denied, 1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(limit), allowed, "Lua atomicity: exactly %d should pass", limit)
	assert.Equal(t, int64(concurrency-limit), denied)
}
