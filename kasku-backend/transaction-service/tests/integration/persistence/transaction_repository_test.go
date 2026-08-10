package persistence_test

import (
	"context"
	"fmt"
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
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s.financial_accounts (id, user_id, name, account_type, balance, initial_balance)
		VALUES ($1, $2, 'Dompet Test', 'CASH', 0, 0)
	`, tenantSchema), accountID, userID)
	require.NoError(t, err)

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
		require.NoError(t, repo.Create(ctx, tenantSchema, userID.String(), tx))

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
		require.NoError(t, repo.Create(ctx, tenantSchema, userID.String(), tx))

		from := time.Now().UTC().AddDate(0, -1, 0)
		to := time.Now().UTC().AddDate(0, 1, 0)
		txs, err := repo.List(ctx, tenantSchema, userID.String(), from, to)
		require.NoError(t, err)
		assert.NotEmpty(t, txs)
	})

	t.Run("SoftDelete — transaksi tidak muncul di GetByID", func(t *testing.T) {
		tx := newTx(25_000, entity.TransactionExpense)
		require.NoError(t, repo.Create(ctx, tenantSchema, userID.String(), tx))

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
		require.NoError(t, repo.Create(ctx, tenantSchema, userID.String(), tx))

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

// GetMonthlyTrend memakai rentang [from, to] eksplisit, bukan "N bulan terakhir" dari NOW().
// Test ini menembak periode yang sepenuhnya di masa lalu — bentuk kueri yang dulu mustahil —
// dan menegaskan batas rentangnya inklusif serta transaksi di luar periode tidak ikut terhitung.
func TestTransactionRepository_GetMonthlyTrend_HonorsExplicitRange(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresTransactionRepository(pool)
	ctx := context.Background()

	accountID := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s.financial_accounts (id, user_id, name, account_type, balance, initial_balance)
		VALUES ($1, $2, 'Dompet Tren', 'CASH', 0, 0)
	`, tenantSchema), accountID, userID)
	require.NoError(t, err)

	seed := func(date time.Time, txType entity.TransactionType, amount int64) {
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.transactions
				(id, sync_id, account_id, transaction_type, amount_idr, transaction_date, is_deleted)
			VALUES ($1, $2, $3, $4, $5, $6, false)
		`, tenantSchema), uuid.New(), uuid.New().String(), accountID, txType, amount, date)
		require.NoError(t, err)
	}

	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	}

	// Di dalam periode Jan–Mar 2024.
	seed(d(2024, time.January, 1), entity.TransactionIncome, 100_000)  // batas awal, inklusif
	seed(d(2024, time.February, 15), entity.TransactionExpense, 40_000)
	seed(d(2024, time.February, 20), entity.TransactionExpense, 10_000) // digabung ke Feb
	seed(d(2024, time.March, 31), entity.TransactionIncome, 70_000)     // batas akhir, inklusif
	// Di luar periode — tidak boleh muncul.
	seed(d(2023, time.December, 31), entity.TransactionIncome, 999_000)
	seed(d(2024, time.April, 1), entity.TransactionExpense, 888_000)

	trend, err := repo.GetMonthlyTrend(ctx, tenantSchema, userID.String(),
		d(2024, time.January, 1), d(2024, time.March, 31))
	require.NoError(t, err)

	require.Len(t, trend, 3, "seharusnya hanya Jan, Feb, Mar 2024")
	assert.Equal(t, "2024-01", trend[0].Month)
	assert.Equal(t, int64(100_000), trend[0].Income)
	assert.Equal(t, int64(0), trend[0].Expense)

	assert.Equal(t, "2024-02", trend[1].Month)
	assert.Equal(t, int64(50_000), trend[1].Expense, "dua pengeluaran Feb harus dijumlahkan")

	assert.Equal(t, "2024-03", trend[2].Month)
	assert.Equal(t, int64(70_000), trend[2].Income)
}
