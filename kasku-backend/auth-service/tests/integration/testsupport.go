// Package integration menyediakan shared helpers untuk integration test
// auth-service yang menggunakan testcontainers.
//
// Versi container di-pin ke versi production (lihat docker-compose.yml):
//   - postgres:16-alpine
//   - redis:7-alpine
//   - rabbitmq:3.13-management-alpine
//
// Container di-cleanup otomatis via t.Cleanup. Tidak ada container reuse —
// setiap test mendapat instance fresh untuk isolation. Trade-off: lambat
// (~5-10s startup per test) vs deterministic state.
package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/persistence"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MigrationsDir mengembalikan path absolut ke direktori migrations/ auth-service.
// Pakai runtime.Caller untuk discover path tanpa bergantung pada cwd.
func MigrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile = .../auth-service/tests/integration/testsupport.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
}

// SetupPostgres spin up postgres:16-alpine container, run migrations, dan
// kembalikan pool + cleanup. Cleanup di-attach via t.Cleanup.
func SetupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pgC, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("kasku_auth_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "failed to start postgres container")
	t.Cleanup(func() {
		_ = pgC.Terminate(context.Background())
	})

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Run migrations menggunakan absolute path
	migrationURL := "file://" + MigrationsDir()
	require.NoError(t, runMigrationsURL(dsn, migrationURL), "failed to run migrations")

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, persistence.PingPostgres(ctx, pool), "pool ping failed")
	return pool
}

// SetupRedis spin up redis:7-alpine container dan kembalikan client.
func SetupRedis(t *testing.T) *redis.Client {
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

	// redis URI: redis://host:port  → ekstrak addr
	opt, err := redis.ParseURL(uri)
	require.NoError(t, err)

	client := redis.NewClient(opt)
	t.Cleanup(func() { _ = client.Close() })

	require.NoError(t, client.Ping(ctx).Err(), "redis ping failed")
	return client
}

// SetupRabbitMQ spin up rabbitmq:3.13-management-alpine + return AMQP connection.
func SetupRabbitMQ(t *testing.T) *amqp.Connection {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	rmqC, err := rabbitmq.Run(ctx,
		"rabbitmq:3.13-management-alpine",
		rabbitmq.WithAdminUsername("test"),
		rabbitmq.WithAdminPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Server startup complete").
				WithStartupTimeout(90*time.Second),
		),
	)
	require.NoError(t, err, "failed to start rabbitmq container")
	t.Cleanup(func() {
		_ = rmqC.Terminate(context.Background())
	})

	uri, err := rmqC.AmqpURL(ctx)
	require.NoError(t, err)

	conn, err := amqp.Dial(uri)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

// runMigrationsURL adalah variant runMigrations yang menerima migration URL
// (mendukung path absolut). RunMigrations di production code pakai relative
// path; di test kita harus pakai absolute karena cwd berbeda.
func runMigrationsURL(dsn, sourceURL string) error {
	// Reuse logic — open migrate dengan URL custom.
	m, err := newMigrate(sourceURL, dsn)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	return m.Up()
}
