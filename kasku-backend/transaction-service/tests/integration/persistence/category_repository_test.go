package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/infrastructure/persistence"
	integration "github.com/TubagusAldiMY/kasku/transaction-service/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryRepository_CRUD(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresCategoryRepository(pool)
	ctx := context.Background()

	newCat := func(name string, catType entity.CategoryType) *entity.Category {
		now := time.Now().UTC()
		return &entity.Category{
			ID:           uuid.New(),
			Name:         name,
			Icon:         "tag",
			Color:        "#6366f1",
			CategoryType: catType,
			IsDefault:    false,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	t.Run("Create dan GetByID", func(t *testing.T) {
		cat := newCat("Makan", entity.CategoryExpense)
		require.NoError(t, repo.Create(ctx, tenantSchema, cat))

		got, err := repo.GetByID(ctx, tenantSchema, cat.ID.String())
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "Makan", got.Name)
		assert.Equal(t, entity.CategoryExpense, got.CategoryType)
	})

	t.Run("GetByID tidak ada — nil dikembalikan", func(t *testing.T) {
		got, err := repo.GetByID(ctx, tenantSchema, uuid.New().String())
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("List — semua kategori aktif dikembalikan", func(t *testing.T) {
		cat := newCat("Transport", entity.CategoryExpense)
		require.NoError(t, repo.Create(ctx, tenantSchema, cat))

		cats, err := repo.List(ctx, tenantSchema)
		require.NoError(t, err)
		assert.NotEmpty(t, cats)
		for _, c := range cats {
			assert.False(t, c.IsDeleted)
		}
	})

	t.Run("Update — nama kategori berubah", func(t *testing.T) {
		cat := newCat("Awal", entity.CategoryBoth)
		require.NoError(t, repo.Create(ctx, tenantSchema, cat))

		cat.Name = "Diupdate"
		cat.CategoryType = entity.CategoryIncome
		require.NoError(t, repo.Update(ctx, tenantSchema, cat))

		got, err := repo.GetByID(ctx, tenantSchema, cat.ID.String())
		require.NoError(t, err)
		assert.Equal(t, "Diupdate", got.Name)
		assert.Equal(t, entity.CategoryIncome, got.CategoryType)
	})

	t.Run("SoftDelete — kategori tidak muncul di List setelah dihapus", func(t *testing.T) {
		cat := newCat("Akan Dihapus", entity.CategoryExpense)
		require.NoError(t, repo.Create(ctx, tenantSchema, cat))

		require.NoError(t, repo.SoftDelete(ctx, tenantSchema, cat.ID.String()))

		got, err := repo.GetByID(ctx, tenantSchema, cat.ID.String())
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("HasActiveTransactions — kategori tanpa transaksi", func(t *testing.T) {
		cat := newCat("Tanpa Transaksi", entity.CategoryExpense)
		require.NoError(t, repo.Create(ctx, tenantSchema, cat))

		has, err := repo.HasActiveTransactions(ctx, tenantSchema, cat.ID.String())
		require.NoError(t, err)
		assert.False(t, has)
	})
}
