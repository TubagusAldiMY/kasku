package cleanup_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/cleanup"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/integration"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedExpiredRefreshToken insert refresh_tokens dengan expires_at di masa lalu.
// CleanupJob menghapus refresh_tokens dengan expires_at < NOW() - 30 hari.
func seedExpiredRefreshToken(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, expiredAgo time.Duration) {
	t.Helper()
	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `
		INSERT INTO public.refresh_tokens (id, user_id, token_hash, expires_at, is_revoked, created_at)
		VALUES ($1, $2, $3, $4, false, $5)
	`, uuid.New(), userID, "h-"+uuid.NewString(), now.Add(-expiredAgo), now.Add(-expiredAgo))
	require.NoError(t, err)
}

func seedFreshRefreshToken(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()
	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `
		INSERT INTO public.refresh_tokens (id, user_id, token_hash, expires_at, is_revoked, created_at)
		VALUES ($1, $2, $3, $4, false, $5)
	`, uuid.New(), userID, "h-"+uuid.NewString(), now.Add(48*time.Hour), now)
	require.NoError(t, err)
}

func seedExpiredEmailVerification(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, expiredAgo time.Duration) {
	t.Helper()
	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `
		INSERT INTO public.email_verifications (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New(), userID, "ev-"+uuid.NewString(), now.Add(-expiredAgo), now.Add(-expiredAgo))
	require.NoError(t, err)
}

func seedPublishedOutbox(t *testing.T, ctx context.Context, pool *pgxpool.Pool, publishedAgo time.Duration) {
	t.Helper()
	now := time.Now().UTC()
	publishedAt := now.Add(-publishedAgo)
	_, err := pool.Exec(ctx, `
		INSERT INTO public.outbox_events (id, event_type, routing_key, payload, created_at, published_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`, uuid.New(), "test.event", "test.route", `{"x":1}`, now.Add(-publishedAgo-time.Hour), publishedAt)
	require.NoError(t, err)
}

func countRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n))
	return n
}

func TestCleanupJob_RunOnce_DeletesExpiredKeepsFresh(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	// Seed user (required for FK)
	u := seedUserCleanup(t, ctx, userRepo, "cl@example.com", "cluser")

	// Refresh tokens: 3 expired (>30 hari), 2 fresh
	for range 3 {
		seedExpiredRefreshToken(t, ctx, pool, u.ID, 35*24*time.Hour)
	}
	for range 2 {
		seedFreshRefreshToken(t, ctx, pool, u.ID)
	}

	// Email verifications: 2 expired (>7 hari), 1 fresh
	seedExpiredEmailVerification(t, ctx, pool, u.ID, 8*24*time.Hour)
	seedExpiredEmailVerification(t, ctx, pool, u.ID, 10*24*time.Hour)
	_, err := pool.Exec(ctx, `
		INSERT INTO public.email_verifications (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, NOW() + INTERVAL '1 hour', NOW())
	`, uuid.New(), u.ID, "ev-fresh")
	require.NoError(t, err)

	// Outbox: 2 published >7 hari, 1 fresh
	seedPublishedOutbox(t, ctx, pool, 8*24*time.Hour)
	seedPublishedOutbox(t, ctx, pool, 9*24*time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO public.outbox_events (id, event_type, routing_key, payload, created_at, published_at)
		VALUES ($1, $2, $3, $4::jsonb, NOW(), NOW())
	`, uuid.New(), "fresh.event", "fresh.route", `{"x":2}`)
	require.NoError(t, err)

	// Pre-check counts
	require.Equal(t, 5, countRows(t, ctx, pool, "refresh_tokens"))
	require.Equal(t, 3, countRows(t, ctx, pool, "email_verifications"))
	require.Equal(t, 3, countRows(t, ctx, pool, "outbox_events"))

	job := cleanup.NewCleanupJob(pool, zerolog.Nop(), time.Hour, false)
	require.NoError(t, job.RunOnce(ctx))

	// Post-check: expired deleted, fresh kept
	assert.Equal(t, 2, countRows(t, ctx, pool, "refresh_tokens"))
	assert.Equal(t, 1, countRows(t, ctx, pool, "email_verifications"))
	assert.Equal(t, 1, countRows(t, ctx, pool, "outbox_events"))
}

func TestCleanupJob_DryRun_LogsButDoesNotDelete(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	u := seedUserCleanup(t, ctx, userRepo, "dr@example.com", "druser")
	for range 5 {
		seedExpiredRefreshToken(t, ctx, pool, u.ID, 31*24*time.Hour)
	}
	require.Equal(t, 5, countRows(t, ctx, pool, "refresh_tokens"))

	job := cleanup.NewCleanupJob(pool, zerolog.Nop(), time.Hour, true) // dryRun=true
	require.NoError(t, job.RunOnce(ctx))

	// Dry-run: tidak boleh ada yang dihapus
	assert.Equal(t, 5, countRows(t, ctx, pool, "refresh_tokens"))
}

func TestCleanupJob_RespectsContextCancel(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)

	u := seedUserCleanup(t, context.Background(), userRepo, "cx@example.com", "cxuser")
	for range 3 {
		seedExpiredRefreshToken(t, context.Background(), pool, u.ID, 31*24*time.Hour)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	job := cleanup.NewCleanupJob(pool, zerolog.Nop(), time.Hour, false)
	// RunOnce harus tetap return tanpa panic (errors di-log internally)
	assert.NoError(t, job.RunOnce(ctx))
}

func TestCleanupJob_PasswordResetUsedAndExpired(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	u := seedUserCleanup(t, ctx, userRepo, "pr@example.com", "pruser")
	now := time.Now().UTC()

	// Token used > 7 hari yang lalu → harus dihapus
	_, err := pool.Exec(ctx, `
		INSERT INTO public.password_reset_tokens (id, user_id, token_hash, expires_at, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), u.ID, "used-old", now.Add(-30*time.Hour), now.Add(-8*24*time.Hour), now.Add(-9*24*time.Hour))
	require.NoError(t, err)

	// Token expired > 7 hari & unused → harus dihapus
	_, err = pool.Exec(ctx, `
		INSERT INTO public.password_reset_tokens (id, user_id, token_hash, expires_at, used_at, created_at)
		VALUES ($1, $2, $3, $4, NULL, $5)
	`, uuid.New(), u.ID, "expired-old", now.Add(-8*24*time.Hour), now.Add(-9*24*time.Hour))
	require.NoError(t, err)

	// Token fresh → kept
	_, err = pool.Exec(ctx, `
		INSERT INTO public.password_reset_tokens (id, user_id, token_hash, expires_at, used_at, created_at)
		VALUES ($1, $2, $3, $4, NULL, NOW())
	`, uuid.New(), u.ID, "fresh", now.Add(1*time.Hour))
	require.NoError(t, err)

	require.Equal(t, 3, countRows(t, ctx, pool, "password_reset_tokens"))

	job := cleanup.NewCleanupJob(pool, zerolog.Nop(), time.Hour, false)
	require.NoError(t, job.RunOnce(ctx))

	assert.Equal(t, 1, countRows(t, ctx, pool, "password_reset_tokens"),
		"only fresh token should remain")
}

// seedUserCleanup adalah helper local — kita tidak share dengan persistence/
// test files (different package).
func seedUserCleanup(t *testing.T, ctx context.Context, repo interface {
	Create(ctx context.Context, u *entity.User) error
}, email, username string) *entity.User {
	t.Helper()
	now := time.Now().UTC()
	u := &entity.User{
		ID:            uuid.New(),
		Email:         email,
		Username:      username,
		PasswordHash:  "$argon2id$...",
		IsActive:      true,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repo.Create(ctx, u))
	return u
}
