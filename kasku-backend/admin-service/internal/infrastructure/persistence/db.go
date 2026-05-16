package persistence

import (
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	migrationSourceURL = "file://migrations"
	maxPoolConnections = 10
)

// NewPostgresPool membuat connection pool pgx untuk DSN yang diberikan.
// Pool ini cocok dipakai sebagai handle ke salah satu dari tiga database yang
// dipakai admin-service: kasku_admin (R/W), kasku_auth (R/W terbatas),
// dan kasku_billing (R/W terbatas).
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

// RunMigrations menjalankan migration kasku_admin (DSN admin-service).
// Hanya dipanggil untuk pool kasku_admin — kasku_auth dan kasku_billing punya migration di service masing-masing.
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

// PingPostgres memeriksa konektivitas pool tertentu untuk health check.
func PingPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
