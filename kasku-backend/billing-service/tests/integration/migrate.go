package integration

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// migrateWrapper adalah thin interface untuk testability.
type migrateWrapper struct {
	m *migrate.Migrate
}

func newMigrate(sourceURL, dsn string) (*migrateWrapper, error) {
	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return nil, err
	}
	return &migrateWrapper{m: m}, nil
}

func (w *migrateWrapper) Up() error {
	if err := w.m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (w *migrateWrapper) Close() (sourceErr, dbErr error) {
	return w.m.Close()
}

// MigrateNoChange ekspos sentinel migrate.ErrNoChange.
var MigrateNoChange = migrate.ErrNoChange
