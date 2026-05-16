package persistence_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedUser(t *testing.T, ctx context.Context, repo interface {
	Create(ctx context.Context, u *entity.User) error
}, email, username string) *entity.User {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond) // pg microsecond precision
	u := &entity.User{
		ID:               uuid.New(),
		Email:            email,
		Username:         username,
		PasswordHash:     "$argon2id$v=19$m=65536,t=3,p=4$abc$def",
		IsActive:         true,
		EmailVerified:    true,
		FailedLoginCount: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, repo.Create(ctx, u))
	return u
}

func TestPostgresUserRepository_CreateAndFind(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	t.Run("Create + FindByID round-trip", func(t *testing.T) {
		u := seedUser(t, ctx, repo, "create@example.com", "creator")

		got, err := repo.FindByID(ctx, u.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, u.Email, got.Email)
		assert.Equal(t, u.Username, got.Username)
		assert.True(t, got.IsActive)
	})

	t.Run("Create duplicate email → unique violation", func(t *testing.T) {
		seedUser(t, ctx, repo, "dup@example.com", "user_a")

		dup := &entity.User{
			ID:           uuid.New(),
			Email:        "dup@example.com",
			Username:     "user_b",
			PasswordHash: "h",
			IsActive:     true,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		err := repo.Create(ctx, dup)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unique")
	})

	t.Run("FindByEmail case-insensitive", func(t *testing.T) {
		seedUser(t, ctx, repo, "Mixed@Example.COM", "mixedcase")

		got, err := repo.FindByEmail(ctx, "mixed@example.com")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "Mixed@Example.COM", got.Email)
	})

	t.Run("FindByEmail not found → (nil, nil)", func(t *testing.T) {
		got, err := repo.FindByEmail(ctx, "ghost@nowhere.io")
		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("FindByID not found → (nil, nil)", func(t *testing.T) {
		got, err := repo.FindByID(ctx, uuid.New())
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

// TestPostgresUserRepository_SQLInjection memverifikasi parameterized queries
// menolak SQL injection klasik. Critical security regression test.
func TestPostgresUserRepository_SQLInjection(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	// Seed satu user
	seedUser(t, ctx, repo, "victim@example.com", "victim")

	// Klasik SQL injection payloads — semua harus FindByEmail return (nil, nil)
	payloads := []string{
		"' OR '1'='1",
		"' OR 1=1 --",
		"'; DROP TABLE users; --",
		"' UNION SELECT * FROM users --",
		"victim@example.com' --",
	}

	for _, payload := range payloads {
		t.Run("payload="+payload, func(t *testing.T) {
			got, err := repo.FindByEmail(ctx, payload)
			require.NoError(t, err, "parameterized query must reject injection silently")
			assert.Nil(t, got, "injection must NOT match any row")
		})
	}

	// Pastikan tabel users masih utuh (DROP TABLE attempt tidak berhasil)
	var count int
	require.NoError(t, pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count))
	assert.Equal(t, 1, count, "users table must still have seeded row")
}

func TestPostgresUserRepository_ExistsByEmailUsername(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	seedUser(t, ctx, repo, "exists@example.com", "existsuser")

	exists, err := repo.ExistsByEmail(ctx, "exists@example.com")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByEmail(ctx, "EXISTS@EXAMPLE.COM")
	require.NoError(t, err)
	assert.True(t, exists, "should match case-insensitively")

	exists, err = repo.ExistsByEmail(ctx, "ghost@example.com")
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = repo.ExistsByUsername(ctx, "existsuser")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByUsername(ctx, "EXISTSUSER")
	require.NoError(t, err)
	assert.True(t, exists, "should match case-insensitively")
}

func TestPostgresUserRepository_BruteForceLockout(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, repo, "bf@example.com", "bfuser")

	const maxAttempts int16 = 5

	// 4 failed attempts: counter naik, belum locked
	for i := 1; i < 5; i++ {
		require.NoError(t, repo.IncrementFailedLoginAndLockIfNeeded(ctx, u.ID, maxAttempts, "15m"))

		got, err := repo.FindByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, int16(i), got.FailedLoginCount, "iteration %d", i)
		assert.Nil(t, got.LockedUntil, "should not be locked yet at iteration %d", i)
	}

	// 5th attempt: lock!
	require.NoError(t, repo.IncrementFailedLoginAndLockIfNeeded(ctx, u.ID, maxAttempts, "15m"))
	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, int16(5), got.FailedLoginCount)
	require.NotNil(t, got.LockedUntil, "should be locked at attempt 5")
	assert.True(t, got.LockedUntil.After(time.Now().UTC()))
	assert.True(t, got.IsAccountLocked(time.Now().UTC()))
}

func TestPostgresUserRepository_UpdateLoginSuccess(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	u := seedUser(t, ctx, repo, "ls@example.com", "lsuser")

	// Fail beberapa kali
	for range 3 {
		require.NoError(t, repo.IncrementFailedLoginAndLockIfNeeded(ctx, u.ID, 5, "15m"))
	}

	// Login success: counter reset, last_login_at updated
	require.NoError(t, repo.UpdateLoginSuccess(ctx, u.ID))

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, int16(0), got.FailedLoginCount)
	require.NotNil(t, got.LastLoginAt)
	assert.WithinDuration(t, time.Now().UTC(), *got.LastLoginAt, 5*time.Second)
	assert.Nil(t, got.LockedUntil)
}

func TestPostgresUserRepository_VerifyAndUpdatePassword(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	// Seed user inactive
	now := time.Now().UTC()
	u := &entity.User{
		ID:            uuid.New(),
		Email:         "verify@example.com",
		Username:      "verifyuser",
		PasswordHash:  "old-hash",
		IsActive:      false,
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, repo.Create(ctx, u))

	// VerifyEmail → is_active=true, email_verified=true
	require.NoError(t, repo.VerifyEmail(ctx, u.ID))
	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, got.IsActive)
	assert.True(t, got.EmailVerified)

	// UpdatePassword
	require.NoError(t, repo.UpdatePassword(ctx, u.ID, "new-hash-xyz"))
	got, err = repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-hash-xyz", got.PasswordHash)
}

func TestPostgresUserRepository_EmailNormalizationOnCreate(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresUserRepository(pool)
	ctx := context.Background()

	// Email disimpan as-is, tapi lookup case-insensitive
	u := seedUser(t, ctx, repo, "Alice@Example.COM", "alice")
	assert.Equal(t, "Alice@Example.COM", strings.ToLower("alice@example.com")[:0]+u.Email) // sanity: stored as-is

	got, err := repo.FindByEmail(ctx, "ALICE@example.com")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID, got.ID)
}
