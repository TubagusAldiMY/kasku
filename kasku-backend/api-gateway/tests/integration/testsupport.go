// Package integration menyediakan shared helpers untuk integration test api-gateway
// menggunakan testcontainers.
package integration

import (
	"context"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupRedis spin up redis:7-alpine dan kembalikan client yang sudah terhubung.
// Container di-cleanup otomatis via t.Cleanup.
func SetupRedis(t *testing.T) *goredis.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	redisC, err := tcredis.Run(ctx, "redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err, "failed to start redis container")
	t.Cleanup(func() {
		_ = redisC.Terminate(context.Background())
	})

	uri, err := redisC.ConnectionString(ctx)
	require.NoError(t, err)

	opt, err := goredis.ParseURL(uri)
	require.NoError(t, err)

	client := goredis.NewClient(opt)
	t.Cleanup(func() { _ = client.Close() })

	require.NoError(t, client.Ping(ctx).Err(), "redis ping failed")
	return client
}
