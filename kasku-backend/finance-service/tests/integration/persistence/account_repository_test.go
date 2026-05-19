package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/infrastructure/persistence"
	integration "github.com/TubagusAldiMY/kasku/finance-service/tests/integration"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountRepository_CRUD(t *testing.T) {
	pool := integration.SetupPostgres(t)
	userID := uuid.New()
	tenantSchema := integration.ProvisionTenant(t, pool, userID.String())
	repo := persistence.NewPostgresAccountRepository(pool)
	ctx := context.Background()

	t.Run("Create dan GetByID", func(t *testing.T) {
		acc := &entity.FinancialAccount{
			ID:          uuid.New(),
			UserID:      userID,
			Name:        "BCA Tabungan",
			AccountType: entity.AccountTypeBank,
			Balance:     1_000_000,
			Currency:    "IDR",
			Color:       "#6366f1",
			Icon:        "bank",
			IsDefault:   true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, tenantSchema, acc))

		got, err := repo.GetByID(ctx, tenantSchema, acc.ID.String(), userID.String())
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, acc.Name, got.Name)
		assert.Equal(t, acc.Balance, got.Balance)
		assert.Equal(t, acc.Currency, got.Currency)
		assert.True(t, got.IsDefault)
	})

	t.Run("GetByID akun tidak ada — ErrAccountNotFound", func(t *testing.T) {
		_, err := repo.GetByID(ctx, tenantSchema, uuid.New().String(), userID.String())
		assert.ErrorIs(t, err, domainerrors.ErrAccountNotFound)
	})

	t.Run("List — hanya akun milik user yang dikembalikan", func(t *testing.T) {
		// Buat akun kedua milik user yang sama
		acc2 := &entity.FinancialAccount{
			ID: uuid.New(), UserID: userID, Name: "OVO Wallet",
			AccountType: entity.AccountTypeEwallet, Balance: 50_000,
			Currency: "IDR", Color: "#7c3aed", Icon: "wallet",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, tenantSchema, acc2))

		accounts, err := repo.List(ctx, tenantSchema, userID.String())
		require.NoError(t, err)
		// Minimal 2 (plus akun dari sub-test sebelumnya)
		assert.GreaterOrEqual(t, len(accounts), 2)
		for _, a := range accounts {
			assert.Equal(t, userID, a.UserID)
			assert.False(t, a.IsDeleted)
		}
	})

	t.Run("CountByUserID — menghitung akun aktif", func(t *testing.T) {
		count, err := repo.CountByUserID(ctx, tenantSchema, userID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 2)
	})

	t.Run("Update — nama dan tipe berubah", func(t *testing.T) {
		acc := &entity.FinancialAccount{
			ID: uuid.New(), UserID: userID, Name: "Awal",
			AccountType: entity.AccountTypeBank, Balance: 0,
			Currency: "IDR", Color: "#000", Icon: "card",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, tenantSchema, acc))

		acc.Name = "Diupdate"
		acc.AccountType = entity.AccountTypeCash
		require.NoError(t, repo.Update(ctx, tenantSchema, acc))

		got, err := repo.GetByID(ctx, tenantSchema, acc.ID.String(), userID.String())
		require.NoError(t, err)
		assert.Equal(t, "Diupdate", got.Name)
		assert.Equal(t, entity.AccountTypeCash, got.AccountType)
	})

	t.Run("SoftDelete — akun tidak muncul di GetByID setelah dihapus", func(t *testing.T) {
		acc := &entity.FinancialAccount{
			ID: uuid.New(), UserID: userID, Name: "Akan Dihapus",
			AccountType: entity.AccountTypeBank, Balance: 0,
			Currency: "IDR", Color: "#f00", Icon: "trash",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, tenantSchema, acc))

		require.NoError(t, repo.SoftDelete(ctx, tenantSchema, acc.ID.String(), userID.String()))

		_, err := repo.GetByID(ctx, tenantSchema, acc.ID.String(), userID.String())
		assert.ErrorIs(t, err, domainerrors.ErrAccountNotFound)
	})

	t.Run("SoftDelete — akun tidak ada — ErrAccountNotFound", func(t *testing.T) {
		err := repo.SoftDelete(ctx, tenantSchema, uuid.New().String(), userID.String())
		assert.ErrorIs(t, err, domainerrors.ErrAccountNotFound)
	})

	t.Run("GetBalanceHistory — tabel kosong mengembalikan slice kosong", func(t *testing.T) {
		acc := &entity.FinancialAccount{
			ID: uuid.New(), UserID: userID, Name: "History Test",
			AccountType: entity.AccountTypeBank, Balance: 0,
			Currency: "IDR", Color: "#000", Icon: "wallet",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, tenantSchema, acc))

		history, err := repo.GetBalanceHistory(ctx, tenantSchema, acc.ID.String(), 3)
		require.NoError(t, err)
		assert.Empty(t, history)
	})
}

func TestAccountRepository_TenantSchemaValidation(t *testing.T) {
	pool := integration.SetupPostgres(t)
	repo := persistence.NewPostgresAccountRepository(pool)
	ctx := context.Background()

	invalidSchemas := []string{
		"public",
		"tenant_abc",
		"tenant-550e8400-e29b-41d4-a716-446655440000",
		"'; DROP TABLE users; --",
		"",
	}

	for _, schema := range invalidSchemas {
		schema := schema
		t.Run("schema tidak valid: "+schema, func(t *testing.T) {
			_, err := repo.CountByUserID(ctx, schema, uuid.New().String())
			assert.Error(t, err, "schema %q seharusnya ditolak", schema)
		})
	}
}
