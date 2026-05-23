//go:build integration

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

func newTestBudget(userID uuid.UUID, name string, limitIDR int64, catID *uuid.UUID) *entity.Budget {
	now := time.Now().UTC()
	return &entity.Budget{
		ID:             uuid.New(),
		UserID:         userID,
		SyncID:         uuid.New().String(),
		Name:           name,
		LimitIDR:       limitIDR,
		CategoryID:     catID,
		PeriodType:     entity.PeriodMonthly,
		StartDate:      now,
		AlertThreshold: 80,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestBudgetRepository_CreateAndList(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresBudgetRepository(pool)
	ctx := context.Background()

	b1 := newTestBudget(userID, "Makan & Minum", 500_000, nil)
	b2 := newTestBudget(userID, "Transport", 200_000, nil)

	require.NoError(t, repo.Create(ctx, tenantSchema, b1))
	require.NoError(t, repo.Create(ctx, tenantSchema, b2))

	list, err := repo.List(ctx, tenantSchema, userID.String())
	require.NoError(t, err)
	assert.Len(t, list, 2)

	names := make([]string, 0, len(list))
	for _, b := range list {
		names = append(names, b.Name)
	}
	assert.Contains(t, names, "Makan & Minum")
	assert.Contains(t, names, "Transport")
}

func TestBudgetRepository_SpendingCalculation(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresBudgetRepository(pool)
	ctx := context.Background()

	accountID := uuid.New()
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s.financial_accounts (id, user_id, name, account_type, balance)
		VALUES ($1, $2, 'Dompet', 'CASH', 1000000)
	`, tenantSchema), accountID, userID)
	require.NoError(t, err)

	b := newTestBudget(userID, "Total Pengeluaran", 500_000, nil)
	require.NoError(t, repo.Create(ctx, tenantSchema, b))

	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s.transactions
			(id, account_id, budget_id, transaction_type, amount_idr, transaction_date)
		VALUES ($1, $2, $3, 'EXPENSE', 150000, CURRENT_DATE)
	`, tenantSchema), uuid.New(), accountID, b.ID)
	require.NoError(t, err)

	list, err := repo.List(ctx, tenantSchema, userID.String())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, int64(150_000), list[0].SpentIDR)
}

func TestBudgetRepository_ExplicitBudgetOnly(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresBudgetRepository(pool)
	ctx := context.Background()

	accountID := uuid.New()
	catA := uuid.New()
	catB := uuid.New()

	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s.financial_accounts (id, user_id, name, account_type, balance)
		VALUES ($1, $2, 'Dompet', 'CASH', 2000000)
	`, tenantSchema), accountID, userID)
	require.NoError(t, err)

	b := newTestBudget(userID, "Semua Pengeluaran", 500_000, nil)
	require.NoError(t, repo.Create(ctx, tenantSchema, b))

	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s.transactions
			(id, account_id, category_id, budget_id, transaction_type, amount_idr, transaction_date)
		VALUES ($1, $2, $3, $4, 'EXPENSE', 100000, CURRENT_DATE)
	`, tenantSchema), uuid.New(), accountID, catA, b.ID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s.transactions
			(id, account_id, category_id, transaction_type, amount_idr, transaction_date)
		VALUES ($1, $2, $3, 'EXPENSE', 75000, CURRENT_DATE)
	`, tenantSchema), uuid.New(), accountID, catB)
	require.NoError(t, err)

	list, err := repo.List(ctx, tenantSchema, userID.String())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, int64(100_000), list[0].SpentIDR)
}

func TestBudgetRepository_TenantIsolation(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userA := uuid.New()
	userB := uuid.New()
	schemaA := integration.ProvisionTenant(t, pool, userA.String())
	schemaB := integration.ProvisionTenant(t, pool, userB.String())
	repo := persistence.NewPostgresBudgetRepository(pool)
	ctx := context.Background()

	b := newTestBudget(userA, "Budget A", 300_000, nil)
	require.NoError(t, repo.Create(ctx, schemaA, b))

	listB, err := repo.List(ctx, schemaB, userB.String())
	require.NoError(t, err)
	assert.Empty(t, listB)
}

func TestBudgetRepository_SoftDelete(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresBudgetRepository(pool)
	ctx := context.Background()

	b := newTestBudget(userID, "Budget Hapus", 100_000, nil)
	require.NoError(t, repo.Create(ctx, tenantSchema, b))

	require.NoError(t, repo.SoftDelete(ctx, tenantSchema, b.ID.String(), userID.String()))

	list, err := repo.List(ctx, tenantSchema, userID.String())
	require.NoError(t, err)
	assert.Empty(t, list)

	// SoftDelete kedua → ErrBudgetNotFound.
	err = repo.SoftDelete(ctx, tenantSchema, b.ID.String(), userID.String())
	require.ErrorIs(t, err, domainerrors.ErrBudgetNotFound)
}

func TestBudgetRepository_Count(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresBudgetRepository(pool)
	ctx := context.Background()

	count, err := repo.Count(ctx, tenantSchema, userID.String())
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, repo.Create(ctx, tenantSchema, newTestBudget(userID, "B1", 200_000, nil)))
	require.NoError(t, repo.Create(ctx, tenantSchema, newTestBudget(userID, "B2", 300_000, nil)))

	count, err = repo.Count(ctx, tenantSchema, userID.String())
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestBudgetRepository_GetByID(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresBudgetRepository(pool)
	ctx := context.Background()

	b := newTestBudget(userID, "Budget Get", 400_000, nil)
	require.NoError(t, repo.Create(ctx, tenantSchema, b))

	got, err := repo.GetByID(ctx, tenantSchema, b.ID.String(), userID.String())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Budget Get", got.Name)
	assert.Equal(t, int64(400_000), got.LimitIDR)

	_, err = repo.GetByID(ctx, tenantSchema, uuid.New().String(), userID.String())
	require.ErrorIs(t, err, domainerrors.ErrBudgetNotFound)
}
