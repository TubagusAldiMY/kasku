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

func TestPostgresPasswordResetRepository_CreateAndFind(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	prRepo, _ := persistence.NewPostgresPasswordResetRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "pr@example.com", "pruser")

	now := time.Now().UTC()
	tok := &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "pr-active",
		ExpiresAt: now.Add(1 * time.Hour),
		CreatedAt: now,
	}
	require.NoError(t, prRepo.Create(ctx, tok))

	got, err := prRepo.FindActiveByTokenHash(ctx, "pr-active")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tok.ID, got.ID)
	assert.Nil(t, got.UsedAt)
}

func TestPostgresPasswordResetRepository_ExpiredExcluded(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	prRepo, _ := persistence.NewPostgresPasswordResetRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "pre@example.com", "preuser")
	now := time.Now().UTC()
	require.NoError(t, prRepo.Create(ctx, &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "pr-expired",
		ExpiresAt: now.Add(-1 * time.Minute),
		CreatedAt: now.Add(-2 * time.Hour),
	}))

	got, err := prRepo.FindActiveByTokenHash(ctx, "pr-expired")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPostgresPasswordResetRepository_UsedExcluded(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	prRepo, _ := persistence.NewPostgresPasswordResetRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "pru@example.com", "pruuser")
	now := time.Now().UTC()
	tok := &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "pr-used",
		ExpiresAt: now.Add(1 * time.Hour),
		CreatedAt: now,
	}
	require.NoError(t, prRepo.Create(ctx, tok))
	require.NoError(t, prRepo.MarkAsUsed(ctx, tok.ID))

	got, err := prRepo.FindActiveByTokenHash(ctx, "pr-used")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// TestPostgresPasswordResetRepository_ExecuteResetPasswordTx_Atomic memverifikasi
// transaksi atomic: SEMUA dari 3 operasi (update password, mark token used,
// revoke refresh tokens) terjadi atau TIDAK ada satupun. Critical security
// requirement — kalau salah satu fail, rollback harus utuh.
func TestPostgresPasswordResetRepository_ExecuteResetPasswordTx_Happy(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	rtRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	prRepo, prTxRepo := persistence.NewPostgresPasswordResetRepository(pool)
	ctx := context.Background()

	// Setup: user + 1 reset token + 2 refresh tokens
	u := seedUser(t, ctx, userRepo, "tx@example.com", "txuser")
	now := time.Now().UTC()
	resetTok := &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "tx-reset",
		ExpiresAt: now.Add(1 * time.Hour),
		CreatedAt: now,
	}
	require.NoError(t, prRepo.Create(ctx, resetTok))
	seedRefreshToken(t, ctx, rtRepo, u.ID, "tx-rt1", time.Hour)
	seedRefreshToken(t, ctx, rtRepo, u.ID, "tx-rt2", time.Hour)

	const newHash = "$argon2id$v=19$m=65536,t=3,p=4$NEW-SALT$NEW-HASH"
	require.NoError(t, prTxRepo.ExecuteResetPasswordTx(ctx, u.ID, newHash, resetTok.ID))

	// Verify 3 effects:
	// 1. Password updated
	gotUser, _ := userRepo.FindByID(ctx, u.ID)
	require.NotNil(t, gotUser)
	assert.Equal(t, newHash, gotUser.PasswordHash)

	// 2. Token marked as used
	gotTok, _ := prRepo.FindActiveByTokenHash(ctx, "tx-reset")
	assert.Nil(t, gotTok, "reset token must be used")

	// 3. Refresh tokens revoked
	for _, h := range []string{"tx-rt1", "tx-rt2"} {
		rt, _ := rtRepo.FindByTokenHash(ctx, h)
		require.NotNil(t, rt)
		assert.True(t, rt.IsRevoked, "refresh token %s must be revoked", h)
	}
}

// TestPostgresPasswordResetRepository_FKCascade memverifikasi DELETE user
// cascade ke password_reset_tokens.
func TestPostgresPasswordResetRepository_FKCascade(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	prRepo, _ := persistence.NewPostgresPasswordResetRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "fkpr@example.com", "fkpruser")
	now := time.Now().UTC()
	require.NoError(t, prRepo.Create(ctx, &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "fk-pr",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}))

	_, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", u.ID)
	require.NoError(t, err)

	got, err := prRepo.FindActiveByTokenHash(ctx, "fk-pr")
	assert.NoError(t, err)
	assert.Nil(t, got)
}
