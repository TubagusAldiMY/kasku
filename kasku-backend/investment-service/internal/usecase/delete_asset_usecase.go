package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/repository"
)

type DeleteAssetUseCase struct {
	repo repository.InvestmentAssetRepository
}

func NewDeleteAssetUseCase(repo repository.InvestmentAssetRepository) *DeleteAssetUseCase {
	return &DeleteAssetUseCase{repo: repo}
}

func (uc *DeleteAssetUseCase) Execute(ctx context.Context, tenantSchema, id string) error {
	return uc.repo.SoftDelete(ctx, tenantSchema, id)
}
