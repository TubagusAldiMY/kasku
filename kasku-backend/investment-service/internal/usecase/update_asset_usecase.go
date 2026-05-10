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

type UpdateAssetInput struct {
	TenantSchema string
	ID           string
	Name         string
	AssetType    entity.AssetType
	Symbol       string
	Currency     string
	SortOrder    int
}

type UpdateAssetUseCase struct {
	repo repository.InvestmentAssetRepository
}

func NewUpdateAssetUseCase(repo repository.InvestmentAssetRepository) *UpdateAssetUseCase {
	return &UpdateAssetUseCase{repo: repo}
}

func (uc *UpdateAssetUseCase) Execute(ctx context.Context, input UpdateAssetInput) (*entity.InvestmentAsset, error) {
	existing, err := uc.repo.GetByID(ctx, input.TenantSchema, input.ID)
	if err != nil {
		return nil, err
	}

	existing.Name = input.Name
	existing.AssetType = input.AssetType
	existing.Symbol = input.Symbol
	existing.Currency = input.Currency
	existing.SortOrder = input.SortOrder

	if err := uc.repo.Update(ctx, input.TenantSchema, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

// RecordUnitChangeInput is the input for recording a BUY/SELL/ADJUST operation.
type RecordUnitChangeInput struct {
	TenantSchema    string
	AssetID         string
	TransactionType string // BUY, SELL, ADJUST
	QuantityChange  float64
	PricePerUnit    float64
	Notes           string
	TransactionDate time.Time
}

// RecordUnitChangeUseCase records a unit change and updates asset quantity.
type RecordUnitChangeUseCase struct {
	repo repository.InvestmentAssetRepository
}

func NewRecordUnitChangeUseCase(repo repository.InvestmentAssetRepository) *RecordUnitChangeUseCase {
	return &RecordUnitChangeUseCase{repo: repo}
}

func (uc *RecordUnitChangeUseCase) Execute(ctx context.Context, input RecordUnitChangeInput) (*entity.UnitHistory, error) {
	// Validate transaction type
	if input.TransactionType != "BUY" && input.TransactionType != "SELL" && input.TransactionType != "ADJUST" {
		return nil, domainerrors.ErrInvalidTransactionType
	}

	// Get current asset
	asset, err := uc.repo.GetByID(ctx, input.TenantSchema, input.AssetID)
	if err != nil {
		return nil, err
	}

	// Calculate new quantity
	newQuantity := asset.Quantity + input.QuantityChange
	if newQuantity < 0 {
		return nil, fmt.Errorf("%w: quantity tidak boleh negatif setelah operasi", domainerrors.ErrInvalidInput)
	}

	// Calculate new average buy price (only for BUY)
	newAvgBuyPrice := asset.AvgBuyPrice
	if input.TransactionType == "BUY" && input.QuantityChange > 0 {
		totalOldValue := asset.Quantity * asset.AvgBuyPrice
		totalNewValue := input.QuantityChange * input.PricePerUnit
		newAvgBuyPrice = (totalOldValue + totalNewValue) / newQuantity
	}

	now := time.Now().UTC()
	entry := &entity.UnitHistory{
		ID:              uuid.New(),
		AssetID:         asset.ID,
		TransactionType: input.TransactionType,
		QuantityChange:  input.QuantityChange,
		PricePerUnit:    input.PricePerUnit,
		Notes:           input.Notes,
		TransactionDate: input.TransactionDate,
		RecordedAt:      now,
	}

	if err := uc.repo.CreateUnitHistory(ctx, input.TenantSchema, entry); err != nil {
		return nil, fmt.Errorf("gagal catat unit history: %w", err)
	}

	if err := uc.repo.UpdateQuantity(ctx, input.TenantSchema, input.AssetID, newQuantity, newAvgBuyPrice); err != nil {
		return nil, fmt.Errorf("gagal update quantity: %w", err)
	}

	return entry, nil
}
