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

// financeServiceMigrationsDir mengembalikan path ke migrations/ finance-service.
// investment-service tidak punya migrations sendiri — tabel investment_assets dan
// unit_history dibuat oleh finance-service via provision_tenant().
func financeServiceMigrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile = .../investment-service/tests/integration/testsupport.go
	// finance-service ada di level yang sama: ../../finance-service/migrations
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "finance-service", "migrations")
	return filepath.Clean(root)
}

// SetupPostgres spin up postgres:16-alpine container, jalankan finance-service migrations
// (yang mencakup provision_tenant + investment_assets + unit_history), dan kembalikan pool.
// Container di-cleanup otomatis via t.Cleanup.
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
	require.NoError(t, err, "gagal start postgres container")
	t.Cleanup(func() {
		_ = pgC.Terminate(context.Background())
	})

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	require.NoError(t, createServiceRoles(ctx, dsn), "gagal create service roles")

	migrationURL := "file://" + financeServiceMigrationsDir()
	require.NoError(t, runMigrations(dsn, migrationURL), "gagal run finance-service migrations")

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx), "pool ping gagal")
	return pool
}

// ProvisionTenant memanggil provision_tenant(user_id::uuid) dari finance-service migration
// untuk membuat tenant schema beserta tabel investment_assets dan unit_history.
func ProvisionTenant(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	_, err := pool.Exec(context.Background(), "SELECT public.provision_tenant($1::uuid)", userID)
	require.NoError(t, err, "gagal provision tenant userID=%s", userID)
	return "tenant_" + strings.ReplaceAll(userID, "-", "_")
}

// createServiceRoles membuat role PostgreSQL yang dibutuhkan oleh GRANT statements
// di dalam finance-service migrations. Di production role ini dibuat oleh
// infra/postgres/00-init-databases.sh; di test container kita buat secara minimal.
func createServiceRoles(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("koneksi untuk create roles: %w", err)
	}
	defer pool.Close()

	roles := []string{
		"kasku_user_svc",
		"kasku_finance_svc",
		"kasku_transaction_svc",
		"kasku_investment_svc",
		"kasku_sync_svc",
	}
	for _, role := range roles {
		if _, err := pool.Exec(ctx, "CREATE ROLE "+role); err != nil {
			// Abaikan error jika role sudah ada
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("create role %s: %w", role, err)
			}
		}
	}
	return nil
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
