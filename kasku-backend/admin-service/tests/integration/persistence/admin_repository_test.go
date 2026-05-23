package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/persistence"
	integration "github.com/TubagusAldiMY/kasku/admin-service/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUserRepository_CRUD(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresAdminUserRepository(pool)
	ctx := context.Background()

	admin := &entity.AdminUser{
		ID:           uuid.New(),
		Username:     "test_admin_" + uuid.New().String()[:8],
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$dGVzdHNhbHQ$dGVzdGhhc2g",
		Role:         entity.AdminRoleSuperAdmin,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	t.Run("CreateBootstrap dan FindByUsername", func(t *testing.T) {
		require.NoError(t, repo.CreateBootstrap(ctx, admin))

		found, err := repo.FindByUsername(ctx, admin.Username)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, admin.ID, found.ID)
		assert.Equal(t, entity.AdminRoleSuperAdmin, found.Role)
	})

	t.Run("FindByID", func(t *testing.T) {
		found, err := repo.FindByID(ctx, admin.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, admin.Username, found.Username)
	})

	t.Run("FindByUsername tidak ada — nil dikembalikan", func(t *testing.T) {
		found, err := repo.FindByUsername(ctx, "nonexistent_admin")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("UpdateLastLogin", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, repo.UpdateLastLogin(ctx, admin.ID, now))

		found, err := repo.FindByID(ctx, admin.ID)
		require.NoError(t, err)
		require.NotNil(t, found.LastLoginAt)
		assert.Equal(t, now.Unix(), found.LastLoginAt.Unix())
	})

	t.Run("Count — minimal 1 admin ada", func(t *testing.T) {
		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
	})
}

func TestAuditLogRepository_CRUD(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresAuditLogRepository(pool)
	adminRepo := persistence.NewPostgresAdminUserRepository(pool)
	ctx := context.Background()

	adminID := uuid.New()
	admin := &entity.AdminUser{
		ID:           adminID,
		Username:     "audit_admin_" + uuid.New().String()[:8],
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$dGVzdHNhbHQ$dGVzdGhhc2g",
		Role:         entity.AdminRoleSuperAdmin,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	require.NoError(t, adminRepo.CreateBootstrap(ctx, admin))

	t.Run("Create dan List", func(t *testing.T) {
		entry := &entity.AuditLogEntry{
			ID:        uuid.New(),
			AdminID:   adminID,
			Action:    entity.AuditActionLogin,
			Metadata:  []byte(`{"ip":"127.0.0.1"}`),
			Success:   true,
			CreatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, entry))

		entries, total, err := repo.List(ctx, repository.AuditLogFilter{Limit: 10})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		assert.NotEmpty(t, entries)
	})
}
