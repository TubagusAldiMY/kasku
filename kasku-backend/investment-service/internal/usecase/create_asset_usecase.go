package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/investment-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/repository"
	"github.com/google/uuid"
)

const defaultCurrency = "IDR"

type CreateAssetInput struct {
	TenantSchema   string
	Name           string
	AssetType      entity.AssetType
	Symbol         string
	Quantity       float64
	AvgBuyPrice    float64
	Currency       string
	MaxInvestments int // dari X-Tier-Max-Investments header; -1 = unlimited
}

type CreateAssetUseCase struct {
	repo repository.InvestmentAssetRepository
}

func NewCreateAssetUseCase(repo repository.InvestmentAssetRepository) *CreateAssetUseCase {
	return &CreateAssetUseCase{repo: repo}
}

func (uc *CreateAssetUseCase) Execute(ctx context.Context, input CreateAssetInput) (*entity.InvestmentAsset, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: nama instrumen wajib diisi", domainerrors.ErrInvalidInput)
	}
	if input.Symbol == "" {
		return nil, fmt.Errorf("%w: symbol wajib diisi", domainerrors.ErrInvalidInput)
	}

	// Cek tier limit hanya jika tidak unlimited
	if input.MaxInvestments >= 0 {
		count, err := uc.repo.CountActive(ctx, input.TenantSchema)
		if err != nil {
			return nil, fmt.Errorf("gagal hitung aset: %w", err)
		}
		if count >= input.MaxInvestments {
			return nil, domainerrors.ErrAssetLimitReached
		}
	}

	if input.Currency == "" {
		input.Currency = defaultCurrency
	}

	now := time.Now().UTC()
	asset := &entity.InvestmentAsset{
		ID:          uuid.New(),
		Name:        input.Name,
		AssetType:   input.AssetType,
		Symbol:      input.Symbol,
		Quantity:    input.Quantity,
		AvgBuyPrice: input.AvgBuyPrice,
		Currency:    input.Currency,
		IsDeleted:   false,
		SortOrder:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.repo.Create(ctx, input.TenantSchema, asset); err != nil {
		return nil, fmt.Errorf("gagal buat aset: %w", err)
	}

	// Record initial unit history if quantity > 0
	if input.Quantity > 0 {
		histEntry := &entity.UnitHistory{
			ID:              uuid.New(),
			AssetID:         asset.ID,
			TransactionType: "BUY",
			QuantityChange:  input.Quantity,
			PricePerUnit:    input.AvgBuyPrice,
			Notes:           "Initial purchase",
			TransactionDate: now,
			RecordedAt:      now,
		}
		if err := uc.repo.CreateUnitHistory(ctx, input.TenantSchema, histEntry); err != nil {
			return nil, fmt.Errorf("gagal catat unit history awal: %w", err)
		}
	}

	return asset, nil
}
