package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	infraredis "github.com/TubagusAldiMY/kasku/api-gateway/internal/infrastructure/redis"
	"github.com/TubagusAldiMY/kasku/api-gateway/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBlacklist_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test (membutuhkan Docker)")
	}

	client := integration.SetupRedis(t)
	bl := infraredis.NewTokenBlacklist(client)
	ctx := context.Background()

	t.Run("JTI baru — tidak ada di blacklist", func(t *testing.T) {
		jti := uuid.New().String()
		blacklisted, err := bl.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.False(t, blacklisted)
	})

	t.Run("JTI di-set Redis — terdeteksi blacklisted", func(t *testing.T) {
		jti := uuid.New().String()
		key := fmt.Sprintf("jti:%s", jti)
		// Simulasi auth-service yang set blacklist saat logout
		err := client.Set(ctx, key, "1", 15*time.Minute).Err()
		require.NoError(t, err)

		blacklisted, err := bl.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.True(t, blacklisted)
	})

	t.Run("JTI yang sudah expire — tidak ada di blacklist lagi", func(t *testing.T) {
		jti := uuid.New().String()
		key := fmt.Sprintf("jti:%s", jti)
		// TTL sangat pendek (1 ms) untuk simulasi expire
		err := client.Set(ctx, key, "1", 1*time.Millisecond).Err()
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		blacklisted, err := bl.IsBlacklisted(ctx, jti)
		require.NoError(t, err)
		assert.False(t, blacklisted, "JTI yang sudah expire seharusnya tidak blacklisted")
	})
}
