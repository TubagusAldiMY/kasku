// Package integration menyediakan shared helpers untuk integration test
// billing-service yang menggunakan testcontainers.
//
// Versi container di-pin ke versi production (lihat docker-compose.yml):
//   - postgres:16-alpine
//   - rabbitmq:3.13-management-alpine
//
// Container di-cleanup otomatis via t.Cleanup. Tidak ada container reuse —
// setiap test mendapat instance fresh untuk isolation.
package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/persistence"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MigrationsDir mengembalikan path absolut ke direktori migrations/ billing-service.
func MigrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile = .../billing-service/tests/integration/testsupport.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
}

// SetupPostgres spin up postgres:16-alpine container, run migrations,
// kembalikan pool + cleanup. Cleanup di-attach via t.Cleanup.
func SetupPostgres(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pgC, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("kasku_billing_test"),
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

	// Pre-create role yang di-grant oleh migration production
	// (kasku_user_svc — cross-DB grant di migration 000003).
	// Tidak butuh password/login privilege di test environment.
	bootstrapPool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	_, err = bootstrapPool.Exec(ctx, `
		DO $$ BEGIN
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kasku_user_svc') THEN
				CREATE ROLE kasku_user_svc;
			END IF;
		END $$;
	`)
	require.NoError(t, err)
	bootstrapPool.Close()

	// Run migrations menggunakan absolute path.
	migrationURL := "file://" + MigrationsDir()
	require.NoError(t, runMigrationsURL(dsn, migrationURL), "failed to run migrations")

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, persistence.PingPostgres(ctx, pool), "pool ping failed")
	return pool, dsn
}

// SetupRabbitMQ spin up rabbitmq:3.13-management-alpine + return AMQP URL & connection.
func SetupRabbitMQ(t *testing.T) (string, *amqp.Connection) {
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

	return uri, conn
}

// runMigrationsURL menjalankan migration up dengan source URL eksplisit.
func runMigrationsURL(dsn, sourceURL string) error {
	m, err := newMigrate(sourceURL, dsn)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	return m.Up()
}
