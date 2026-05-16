package cleanup_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/cleanup"
	"github.com/TubagusAldiMY/kasku/billing-service/tests/integration"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedPublishedOutbox inserts row dengan published_at di masa lalu.
func seedPublishedOutbox(t *testing.T, pool *pgxpool.Pool, age time.Duration) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO public.outbox_events (id, event_type, routing_key, payload, published_at, publish_attempts)
		VALUES ($1, $2, $3, $4::jsonb, NOW() - $5::interval, 1)
	`, id, "subscription.expired", "subscription.expired", `{}`, age.String())
	require.NoError(t, err)
	return id
}

// seedUnpublishedOutbox inserts row tanpa published_at — tidak boleh terhapus.
func seedUnpublishedOutbox(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO public.outbox_events (id, event_type, routing_key, payload, published_at)
		VALUES ($1, $2, $3, $4::jsonb, NULL)
	`, id, "subscription.expired", "subscription.expired", `{}`)
	require.NoError(t, err)
	return id
}

func countOutbox(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM public.outbox_events`).Scan(&n))
	return n
}

func TestCleanupJob_DeletesOldPublishedOutbox(t *testing.T) {
	pool, _ := integration.SetupPostgres(t)

	oldID := seedPublishedOutbox(t, pool, 10*24*time.Hour) // 10 hari → harus terhapus
	freshID := seedPublishedOutbox(t, pool, 1*time.Hour)   // 1 jam → harus tetap
	unpubID := seedUnpublishedOutbox(t, pool)              // belum publish → harus tetap

	job := cleanup.NewCleanupJob(pool, zerolog.Nop(), time.Hour, false)
	require.NoError(t, job.RunOnce(context.Background()))

	// Old removed, fresh + unpublished kept.
	rows := []struct {
		id     uuid.UUID
		exists bool
	}{
		{oldID, false},
		{freshID, true},
		{unpubID, true},
	}
	for _, r := range rows {
		var n int
		require.NoError(t, pool.QueryRow(context.Background(),
			`SELECT count(*) FROM public.outbox_events WHERE id = $1`, r.id).Scan(&n))
		if r.exists {
			assert.Equal(t, 1, n, "row %s should exist", r.id)
		} else {
			assert.Equal(t, 0, n, "row %s should be deleted", r.id)
		}
	}
}

func TestCleanupJob_DryRun_DoesNotDelete(t *testing.T) {
	pool, _ := integration.SetupPostgres(t)
	seedPublishedOutbox(t, pool, 10*24*time.Hour)
	before := countOutbox(t, pool)

	job := cleanup.NewCleanupJob(pool, zerolog.Nop(), time.Hour, true) // dryRun=true
	require.NoError(t, job.RunOnce(context.Background()))

	after := countOutbox(t, pool)
	assert.Equal(t, before, after, "dry run must not delete")
}
