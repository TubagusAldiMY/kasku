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

func TestPostgresEmailVerificationRepository_CreateAndFind(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	evRepo := persistence.NewPostgresEmailVerificationRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "ev@example.com", "evuser")

	now := time.Now().UTC()
	verif := &entity.EmailVerification{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "ev-hash-active",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}
	require.NoError(t, evRepo.Create(ctx, verif))

	got, err := evRepo.FindActiveByTokenHash(ctx, "ev-hash-active")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, verif.ID, got.ID)
	assert.Nil(t, got.VerifiedAt)
}

func TestPostgresEmailVerificationRepository_ExpiredExcluded(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	evRepo := persistence.NewPostgresEmailVerificationRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "exp@example.com", "expuser")

	now := time.Now().UTC()
	expired := &entity.EmailVerification{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "ev-expired",
		ExpiresAt: now.Add(-1 * time.Hour),
		CreatedAt: now.Add(-25 * time.Hour),
	}
	require.NoError(t, evRepo.Create(ctx, expired))

	got, err := evRepo.FindActiveByTokenHash(ctx, "ev-expired")
	require.NoError(t, err)
	assert.Nil(t, got, "expired token must NOT be returned by FindActiveByTokenHash")
}

func TestPostgresEmailVerificationRepository_VerifiedExcluded(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	evRepo := persistence.NewPostgresEmailVerificationRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "vr@example.com", "vruser")

	now := time.Now().UTC()
	verif := &entity.EmailVerification{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "ev-verified",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}
	require.NoError(t, evRepo.Create(ctx, verif))

	// Mark as verified
	require.NoError(t, evRepo.MarkAsVerified(ctx, verif.ID))

	got, err := evRepo.FindActiveByTokenHash(ctx, "ev-verified")
	require.NoError(t, err)
	assert.Nil(t, got, "verified token must NOT be returned")
}

func TestPostgresEmailVerificationRepository_InvalidateAllActiveByUserID(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	evRepo := persistence.NewPostgresEmailVerificationRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "inv@example.com", "invuser")

	now := time.Now().UTC()
	for i, hash := range []string{"a", "b", "c"} {
		require.NoError(t, evRepo.Create(ctx, &entity.EmailVerification{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: hash,
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}))
	}

	require.NoError(t, evRepo.InvalidateAllActiveByUserID(ctx, u.ID))

	// Semua token harus jadi tidak aktif
	for _, h := range []string{"a", "b", "c"} {
		got, _ := evRepo.FindActiveByTokenHash(ctx, h)
		assert.Nil(t, got, "token %s should be invalidated", h)
	}
}

func TestPostgresEmailVerificationRepository_FKCascade(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userRepo := persistence.NewPostgresUserRepository(pool)
	evRepo := persistence.NewPostgresEmailVerificationRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, userRepo, "fkev@example.com", "fkevuser")
	now := time.Now().UTC()
	require.NoError(t, evRepo.Create(ctx, &entity.EmailVerification{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "fk-ev",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}))

	// Delete user → email_verifications harus cascade
	_, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", u.ID)
	require.NoError(t, err)

	got, err := evRepo.FindActiveByTokenHash(ctx, "fk-ev")
	assert.NoError(t, err)
	assert.Nil(t, got)
}
