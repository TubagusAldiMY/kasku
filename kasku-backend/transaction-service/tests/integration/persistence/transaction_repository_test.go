package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/infrastructure/persistence"
	integration "github.com/TubagusAldiMY/kasku/transaction-service/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionRepository_CRUD(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresTransactionRepository(pool)
	ctx := context.Background()

	accountID := uuid.New()

	newTx := func(amount int64, txType entity.TransactionType) *entity.Transaction {
		now := time.Now().UTC()
		return &entity.Transaction{
			ID:              uuid.New(),
			SyncID:          uuid.New().String(),
			AccountID:       accountID,
			TransactionType: txType,
			AmountIDR:       amount,
			TransactionDate: now,
			Notes:           "test",
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}

	t.Run("Create dan GetByID", func(t *testing.T) {
		tx := newTx(100_000, entity.TransactionIncome)
		require.NoError(t, repo.Create(ctx, tenantSchema, tx))

		got, err := repo.GetByID(ctx, tenantSchema, tx.ID.String(), userID.String())
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, int64(100_000), got.AmountIDR)
		assert.Equal(t, entity.TransactionIncome, got.TransactionType)
	})

	t.Run("GetByID transaksi tidak ada — nil dikembalikan", func(t *testing.T) {
		got, err := repo.GetByID(ctx, tenantSchema, uuid.New().String(), userID.String())
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("List — mengembalikan transaksi dalam rentang waktu", func(t *testing.T) {
		tx := newTx(50_000, entity.TransactionExpense)
		require.NoError(t, repo.Create(ctx, tenantSchema, tx))

		from := time.Now().UTC().AddDate(0, -1, 0)
		to := time.Now().UTC().AddDate(0, 1, 0)
		txs, err := repo.List(ctx, tenantSchema, userID.String(), from, to)
		require.NoError(t, err)
		assert.NotEmpty(t, txs)
	})

	t.Run("SoftDelete — transaksi tidak muncul di GetByID", func(t *testing.T) {
		tx := newTx(25_000, entity.TransactionExpense)
		require.NoError(t, repo.Create(ctx, tenantSchema, tx))

		require.NoError(t, repo.SoftDelete(ctx, tenantSchema, tx.ID.String(), userID.String()))

		got, err := repo.GetByID(ctx, tenantSchema, tx.ID.String(), userID.String())
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("SoftDelete transaksi tidak ada — ErrTransactionNotFound", func(t *testing.T) {
		err := repo.SoftDelete(ctx, tenantSchema, uuid.New().String(), userID.String())
		assert.ErrorIs(t, err, domainerrors.ErrTransactionNotFound)
	})

	t.Run("ListForExport — semua transaksi user dikembalikan", func(t *testing.T) {
		tx := newTx(200_000, entity.TransactionIncome)
		require.NoError(t, repo.Create(ctx, tenantSchema, tx))

		all, err := repo.ListForExport(ctx, tenantSchema, userID.String(), nil, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, all)
	})
}

func TestTransactionRepository_TenantSchemaValidation(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresTransactionRepository(pool)
	ctx := context.Background()

	invalidSchemas := []string{"public", "tenant_abc", "", "'; DROP TABLE transactions; --"}
	for _, schema := range invalidSchemas {
		schema := schema
		t.Run("schema tidak valid: "+schema, func(t *testing.T) {
			_, err := repo.GetByID(ctx, schema, uuid.New().String(), uuid.New().String())
			assert.Error(t, err)
		})
	}
}
