package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	migrationSourceURL = "file://migrations"
	maxPoolConnections = 20
)

// NewPostgresPool membuat koneksi pool PostgreSQL menggunakan pgxpool.
// Pool dikonfigurasi dengan batas koneksi yang wajar untuk lingkungan microservice.
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

// RunMigrations menjalankan semua migration yang belum dijalankan (up).
// Migration files dibaca dari direktori ./migrations.
func RunMigrations(dsn string) (err error) {
	m, err := migrate.New(migrationSourceURL, dsn)
	if err != nil {
		return fmt.Errorf("gagal inisialisasi migrate: %w", err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		err = errors.Join(
			err,
			wrapMigrationCloseError("gagal tutup migration source", sourceErr),
			wrapMigrationCloseError("gagal tutup migration database", databaseErr),
		)
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("gagal menjalankan migration: %w", err)
	}

	return nil
}

func wrapMigrationCloseError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// PingPostgres memeriksa koneksi ke PostgreSQL untuk health check.
func PingPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
