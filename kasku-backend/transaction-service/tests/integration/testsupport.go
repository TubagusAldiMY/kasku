// Package integration menyediakan shared helpers untuk integration test transaction-service.
// Tenant schema dibuat langsung (tanpa memanggil provision_tenant finance-service) untuk
// menjaga isolasi antar service.
package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupPostgres spin up postgres:16-alpine container dengan schema transaksi.
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

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx), "pool ping failed")
	return pool
}

// ProvisionTenant membuat schema dan tabel tenant secara langsung (DDL inline)
// tanpa bergantung pada stored function provision_tenant milik finance-service.
func ProvisionTenant(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	ctx := context.Background()

	tenantSchema := "tenant_" + strings.ReplaceAll(userID, "-", "_")

	sqls := []string{
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, tenantSchema),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.transactions (
				id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
				sync_id           VARCHAR(100) UNIQUE,
				account_id        UUID         NOT NULL,
				category_id       UUID         NULL,
				transaction_type  VARCHAR(20)  NOT NULL,
				amount_idr        BIGINT       NOT NULL,
				transaction_date  DATE         NOT NULL DEFAULT CURRENT_DATE,
				notes             TEXT         NULL,
				to_account_id     UUID         NULL,
				is_deleted        BOOLEAN      NOT NULL DEFAULT false,
				deleted_at        TIMESTAMPTZ  NULL,
				created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
				updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
			)`, tenantSchema),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.categories (
				id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
				name           VARCHAR(50)  NOT NULL,
				icon           VARCHAR(50)  NOT NULL DEFAULT 'tag',
				color          VARCHAR(7)   NOT NULL DEFAULT '#6366f1',
				category_type  VARCHAR(20)  NOT NULL DEFAULT 'BOTH',
				is_default     BOOLEAN      NOT NULL DEFAULT false,
				is_deleted     BOOLEAN      NOT NULL DEFAULT false,
				deleted_at     TIMESTAMPTZ  NULL,
				created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
				updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
			)`, tenantSchema),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.financial_accounts (
				id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id      UUID         NOT NULL,
				name         VARCHAR(100) NOT NULL,
				account_type VARCHAR(30)  NOT NULL DEFAULT 'CASH',
				balance      BIGINT       NOT NULL DEFAULT 0,
				currency     VARCHAR(10)  NOT NULL DEFAULT 'IDR',
				is_deleted   BOOLEAN      NOT NULL DEFAULT false,
				deleted_at   TIMESTAMPTZ  NULL,
				created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
				updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
			)`, tenantSchema),
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s.budgets (
				id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id         UUID         NOT NULL,
				sync_id         VARCHAR(100) UNIQUE,
				name            VARCHAR(100) NOT NULL,
				limit_idr       BIGINT       NOT NULL CHECK (limit_idr > 0),
				category_id     UUID         NULL,
				period_type     VARCHAR(20)  NOT NULL DEFAULT 'MONTHLY',
				start_date      DATE         NOT NULL DEFAULT CURRENT_DATE,
				end_date        DATE         NULL,
				alert_threshold SMALLINT     NOT NULL DEFAULT 80 CHECK (alert_threshold BETWEEN 0 AND 100),
				is_deleted      BOOLEAN      NOT NULL DEFAULT false,
				deleted_at      TIMESTAMPTZ  NULL,
				created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
				updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
			)`, tenantSchema),
	}

	for _, sql := range sqls {
		_, err := pool.Exec(ctx, sql)
		require.NoError(t, err, "failed to provision tenant DDL: %s", sql[:40])
	}

	return tenantSchema
}
