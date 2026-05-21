package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/investment-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/infrastructure/persistence"
	integration "github.com/TubagusAldiMY/kasku/investment-service/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAsset(userID uuid.UUID) *entity.InvestmentAsset {
	return &entity.InvestmentAsset{
		ID:          uuid.New(),
		Name:        "Bitcoin",
		AssetType:   entity.AssetTypeCrypto,
		Symbol:      "BTC",
		Quantity:    0.5,
		AvgBuyPrice: 800_000_000,
		Currency:    "IDR",
		SortOrder:   1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

func TestInvestmentRepository_CreateAndGetByID(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresInvestmentRepository(pool)
	ctx := context.Background()

	asset := newTestAsset(userID)
	require.NoError(t, repo.Create(ctx, tenantSchema, asset))

	got, err := repo.GetByID(ctx, tenantSchema, asset.ID.String())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, asset.ID, got.ID)
	assert.Equal(t, asset.Name, got.Name)
	assert.Equal(t, asset.Symbol, got.Symbol)
	assert.Equal(t, asset.Quantity, got.Quantity)
	assert.Equal(t, asset.AvgBuyPrice, got.AvgBuyPrice)
	assert.False(t, got.IsDeleted)
}

func TestInvestmentRepository_GetByID_NotFound(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresInvestmentRepository(pool)

	_, err := repo.GetByID(context.Background(), tenantSchema, uuid.New().String())
	assert.ErrorIs(t, err, domainerrors.ErrAssetNotFound)
}

func TestInvestmentRepository_List_ExcludesSoftDeleted(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresInvestmentRepository(pool)
	ctx := context.Background()

	a1 := newTestAsset(userID)
	a2 := newTestAsset(userID)
	a2.Name = "Ethereum"
	a2.Symbol = "ETH"
	a2.SortOrder = 2
	require.NoError(t, repo.Create(ctx, tenantSchema, a1))
	require.NoError(t, repo.Create(ctx, tenantSchema, a2))

	// Soft-delete a1
	require.NoError(t, repo.SoftDelete(ctx, tenantSchema, a1.ID.String()))

	assets, err := repo.List(ctx, tenantSchema)
	require.NoError(t, err)
	assert.Len(t, assets, 1)
	assert.Equal(t, a2.ID, assets[0].ID)
}

func TestInvestmentRepository_TenantIsolation(t *testing.T) {
	pool := integration.SetupPostgres(t)
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()
	tenantA := integration.ProvisionTenant(t, pool, userA.String())
	tenantB := integration.ProvisionTenant(t, pool, userB.String())
	repo := persistence.NewPostgresInvestmentRepository(pool)

	assetA := newTestAsset(userA)
	require.NoError(t, repo.Create(ctx, tenantA, assetA))

	// Tenant B tidak boleh lihat aset tenant A
	_, err := repo.GetByID(ctx, tenantB, assetA.ID.String())
	assert.ErrorIs(t, err, domainerrors.ErrAssetNotFound)

	listB, err := repo.List(ctx, tenantB)
	require.NoError(t, err)
	assert.Empty(t, listB)
}

func TestInvestmentRepository_SoftDelete_AlreadyDeleted(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresInvestmentRepository(pool)
	ctx := context.Background()

	asset := newTestAsset(userID)
	require.NoError(t, repo.Create(ctx, tenantSchema, asset))
	require.NoError(t, repo.SoftDelete(ctx, tenantSchema, asset.ID.String()))

	// Soft-delete kedua kali harus ErrAssetNotFound
	err := repo.SoftDelete(ctx, tenantSchema, asset.ID.String())
	assert.ErrorIs(t, err, domainerrors.ErrAssetNotFound)
}

func TestInvestmentRepository_CreateUnitHistory_AndGet(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresInvestmentRepository(pool)
	ctx := context.Background()

	asset := newTestAsset(userID)
	require.NoError(t, repo.Create(ctx, tenantSchema, asset))

	entry := &entity.UnitHistory{
		ID:              uuid.New(),
		AssetID:         asset.ID,
		TransactionType: "BUY",
		QuantityChange:  0.5,
		PricePerUnit:    800_000_000,
		Notes:           "pembelian awal",
		TransactionDate: time.Now().UTC(),
		RecordedAt:      time.Now().UTC(),
	}
	require.NoError(t, repo.CreateUnitHistory(ctx, tenantSchema, entry))

	history, err := repo.GetUnitHistory(ctx, tenantSchema, asset.ID.String(), 0)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, entry.ID, history[0].ID)
	assert.Equal(t, "BUY", history[0].TransactionType)
	assert.Equal(t, 0.5, history[0].QuantityChange)
	assert.InDelta(t, 400_000_000.0, history[0].TotalValue, 0.01)
}

func TestInvestmentRepository_CountActive(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresInvestmentRepository(pool)
	ctx := context.Background()

	count, err := repo.CountActive(ctx, tenantSchema)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, repo.Create(ctx, tenantSchema, newTestAsset(userID)))
	require.NoError(t, repo.Create(ctx, tenantSchema, func() *entity.InvestmentAsset {
		a := newTestAsset(userID)
		a.Symbol = "ETH"
		return a
	}()))

	count, err = repo.CountActive(ctx, tenantSchema)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
