package persistence_test

import (
	"context"
	"testing"

	"github.com/TubagusAldiMY/kasku/user-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/user-service/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserProfileRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test (membutuhkan Docker)")
	}

	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresUserProfileRepository(pool)
	ctx := context.Background()

	t.Run("EnsureUserProfile — insert baru berhasil", func(t *testing.T) {
		userID := uuid.New().String()
		email := "test@example.com"
		username := "testuser"

		err := repo.EnsureUserProfile(ctx, userID, email, username)
		require.NoError(t, err)

		profile, err := repo.GetUserProfile(ctx, userID)
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, email, profile.Email)
		assert.Equal(t, username, profile.Username)
	})

	t.Run("EnsureUserProfile — idempotent pada conflict", func(t *testing.T) {
		userID := uuid.New().String()
		email := "dup@example.com"
		username := "dupuser"

		require.NoError(t, repo.EnsureUserProfile(ctx, userID, email, username))
		// Panggil kedua kali dengan email baru — harus update tanpa error
		require.NoError(t, repo.EnsureUserProfile(ctx, userID, "updated@example.com", username))

		profile, err := repo.GetUserProfile(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", profile.Email)
	})

	t.Run("GetUserProfile — user tidak ada mengembalikan nil", func(t *testing.T) {
		nonExistent := uuid.New().String()
		profile, err := repo.GetUserProfile(ctx, nonExistent)
		require.NoError(t, err)
		assert.Nil(t, profile)
	})

	t.Run("UpdateUserProfile — username dan display_name terupdate", func(t *testing.T) {
		userID := uuid.New().String()
		require.NoError(t, repo.EnsureUserProfile(ctx, userID, "up@example.com", "oldname"))

		updated, err := repo.UpdateUserProfile(ctx, userID, "newname", "New Display")
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "newname", updated.Username)
		require.NotNil(t, updated.DisplayName)
		assert.Equal(t, "New Display", *updated.DisplayName)
	})

	t.Run("UpdateUserProfile — user tidak ada mengembalikan nil", func(t *testing.T) {
		nonExistent := uuid.New().String()
		updated, err := repo.UpdateUserProfile(ctx, nonExistent, "name", "display")
		require.NoError(t, err)
		assert.Nil(t, updated)
	})
}
