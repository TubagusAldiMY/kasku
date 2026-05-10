package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/repository"
)

type ListAssetsUseCase struct {
	repo repository.InvestmentAssetRepository
}

func NewListAssetsUseCase(repo repository.InvestmentAssetRepository) *ListAssetsUseCase {
	return &ListAssetsUseCase{repo: repo}
}

func (uc *ListAssetsUseCase) Execute(ctx context.Context, tenantSchema string) ([]entity.InvestmentAsset, error) {
	return uc.repo.List(ctx, tenantSchema)
}
