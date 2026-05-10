package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/repository"
)

type GetAssetUseCase struct {
	repo repository.InvestmentAssetRepository
}

func NewGetAssetUseCase(repo repository.InvestmentAssetRepository) *GetAssetUseCase {
	return &GetAssetUseCase{repo: repo}
}

func (uc *GetAssetUseCase) Execute(ctx context.Context, tenantSchema, id string) (*entity.InvestmentAsset, error) {
	return uc.repo.GetByID(ctx, tenantSchema, id)
}
