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
	maxPoolConnections = int32(20)
)

// NewPostgresPool membuat connection pool baru ke PostgreSQL.
// Akan mengembalikan error jika DSN tidak valid atau database tidak terjangkau.
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

// RunMigrations menjalankan semua migration yang belum diaplikasikan.
// Dijalankan saat startup sebelum server menerima traffic.
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

// PingPostgres memeriksa konektivitas ke PostgreSQL — digunakan oleh health check endpoint.
func PingPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
