package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedRefreshToken(t *testing.T, ctx context.Context, repo interface {
	Create(ctx context.Context, t *entity.RefreshToken) error
}, userID uuid.UUID, hash string, expiresIn time.Duration) *entity.RefreshToken {
	t.Helper()
	now := time.Now().UTC()
	ua := "test-agent"
	ip := "127.0.0.1"
	tok := &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hash,
		UserAgent: &ua,
		IPAddress: &ip,
		ExpiresAt: now.Add(expiresIn),
		CreatedAt: now,
	}
	require.NoError(t, repo.Create(ctx, tok))
	return tok
}

func TestPostgresRefreshTokenRepository_CreateAndFind(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	tokRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "rt@example.com", "rtuser")
	tok := seedRefreshToken(t, ctx, tokRepo, u.ID, "hash-abc", 24*time.Hour)

	got, err := tokRepo.FindByTokenHash(ctx, "hash-abc")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tok.ID, got.ID)
	assert.Equal(t, u.ID, got.UserID)
	assert.False(t, got.IsRevoked)
	require.NotNil(t, got.UserAgent)
	assert.Equal(t, "test-agent", *got.UserAgent)
	require.NotNil(t, got.IPAddress)
}

func TestPostgresRefreshTokenRepository_FindNotFound(t *testing.T) {
	pool := integration.SetupPostgres(t)
	tokRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	ctx := context.Background()

	got, err := tokRepo.FindByTokenHash(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestPostgresRefreshTokenRepository_RevokeByID(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	tokRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "rev@example.com", "revuser")
	tok := seedRefreshToken(t, ctx, tokRepo, u.ID, "h1", time.Hour)

	require.NoError(t, tokRepo.RevokeByID(ctx, tok.ID))

	got, err := tokRepo.FindByTokenHash(ctx, "h1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.IsRevoked)
	require.NotNil(t, got.RevokedAt)
}

func TestPostgresRefreshTokenRepository_RevokeAllActiveByUserID(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	tokRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "multi@example.com", "multiuser")
	seedRefreshToken(t, ctx, tokRepo, u.ID, "h-a", time.Hour)
	seedRefreshToken(t, ctx, tokRepo, u.ID, "h-b", time.Hour)
	seedRefreshToken(t, ctx, tokRepo, u.ID, "h-c", time.Hour)

	// Other user's token tetap aktif
	otherU := seedUser(t, ctx, userRepo, "other@example.com", "otheruser")
	seedRefreshToken(t, ctx, tokRepo, otherU.ID, "h-other", time.Hour)

	require.NoError(t, tokRepo.RevokeAllActiveByUserID(ctx, u.ID))

	for _, h := range []string{"h-a", "h-b", "h-c"} {
		got, _ := tokRepo.FindByTokenHash(ctx, h)
		require.NotNil(t, got)
		assert.True(t, got.IsRevoked, "token %s should be revoked", h)
	}

	// Other user's token tidak terpengaruh
	otherTok, _ := tokRepo.FindByTokenHash(ctx, "h-other")
	require.NotNil(t, otherTok)
	assert.False(t, otherTok.IsRevoked)
}

func TestPostgresRefreshTokenRepository_FKCascadeOnUserDelete(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	tokRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "fk@example.com", "fkuser")
	seedRefreshToken(t, ctx, tokRepo, u.ID, "fk-hash", time.Hour)

	// Delete user
	_, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", u.ID)
	require.NoError(t, err)

	// Refresh token should be cascade-deleted
	got, err := tokRepo.FindByTokenHash(ctx, "fk-hash")
	assert.NoError(t, err)
	assert.Nil(t, got, "refresh_tokens should cascade delete when user is deleted")
}

func TestPostgresRefreshTokenRepository_RevokeAllByUserIDInTx(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	tokRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "tx@example.com", "txuser")
	seedRefreshToken(t, ctx, tokRepo, u.ID, "tx-h", time.Hour)

	// Commit path
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tokRepo.RevokeAllByUserIDInTx(ctx, tx, u.ID))
	require.NoError(t, tx.Commit(ctx))

	got, _ := tokRepo.FindByTokenHash(ctx, "tx-h")
	require.NotNil(t, got)
	assert.True(t, got.IsRevoked)

	// Rollback path
	seedRefreshToken(t, ctx, tokRepo, u.ID, "tx-rollback", time.Hour)
	tx2, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tokRepo.RevokeAllByUserIDInTx(ctx, tx2, u.ID))
	require.NoError(t, tx2.Rollback(ctx))

	gotRollback, _ := tokRepo.FindByTokenHash(ctx, "tx-rollback")
	require.NotNil(t, gotRollback)
	assert.False(t, gotRollback.IsRevoked, "rollback must revert revocation")
}

func TestPostgresRefreshTokenRepository_RevokeAllByUserIDInTx_InvalidTx(t *testing.T) {
	pool := integration.SetupPostgres(t)
	tokRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	ctx := context.Background()

	// Pass non-pgx.Tx → harus error
	err := tokRepo.RevokeAllByUserIDInTx(ctx, "not-a-tx", uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaksi tidak valid")
}
