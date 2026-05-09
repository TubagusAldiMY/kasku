package persistence

import (
	"context"
	"fmt"
	"regexp"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	migrationSourceURL = "file://migrations"
	maxPoolConnections = 20
)

// tenantSchemaRegex memvalidasi format tenant schema untuk mencegah SQL injection.
// Format yang valid: tenant_550e8400_e29b_41d4_a716_446655440000
var tenantSchemaRegex = regexp.MustCompile(`^tenant_[0-9a-f_]{32,36}$`)

// ValidateTenantSchema memvalidasi format tenant schema sebelum digunakan dalam query.
// WAJIB dipanggil di setiap repository method sebelum interpolasi nama schema ke SQL.
func ValidateTenantSchema(schema string) error {
	if !tenantSchemaRegex.MatchString(schema) {
		return fmt.Errorf("format tenant schema tidak valid: %s", schema)
	}
	return nil
}

func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("gagal parse PostgreSQL DSN: %w", err)
	}
	poolConfig.MaxConns = maxPoolConnections

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat PostgreSQL connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("gagal ping PostgreSQL: %w", err)
	}

	return pool, nil
}

func RunMigrations(dsn string) error {
	m, err := migrate.New(migrationSourceURL, dsn)
	if err != nil {
		return fmt.Errorf("gagal inisialisasi migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("gagal menjalankan migration: %w", err)
	}
	return nil
}

func PingPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
