package integration

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// migrationsDir mengembalikan path absolut ke direktori migrations/ finance-service.
func migrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile = .../finance-service/tests/integration/testsupport.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
}

// SetupPostgres spin up postgres:16-alpine container, run migrations, dan kembalikan pool.
// Container dan pool di-cleanup otomatis via t.Cleanup.
func SetupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pgC, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("kasku_finance_test"),
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

	migrationURL := "file://" + migrationsDir()
	require.NoError(t, runMigrations(dsn, migrationURL), "failed to run finance-service migrations")

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx), "pool ping failed")
	return pool
}

// ProvisionTenant memanggil stored function provision_tenant(user_id::uuid) untuk
// membuat schema dan tabel tenant. Fungsi ini dibuat oleh migration finance-service.
func ProvisionTenant(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, "SELECT public.provision_tenant($1::uuid)", userID)
	require.NoError(t, err, "failed to provision tenant for userID=%s", userID)

	// tenant schema name: tenant_<uuid-with-underscores>
	return "tenant_" + strings.ReplaceAll(userID, "-", "_")
}

func runMigrations(dsn, sourceURL string) error {
	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
