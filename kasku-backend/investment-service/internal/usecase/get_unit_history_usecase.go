package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/repository"
)

type GetUnitHistoryUseCase struct {
	repo repository.InvestmentAssetRepository
}

func NewGetUnitHistoryUseCase(repo repository.InvestmentAssetRepository) *GetUnitHistoryUseCase {
	return &GetUnitHistoryUseCase{repo: repo}
}

func (uc *GetUnitHistoryUseCase) Execute(ctx context.Context, tenantSchema, assetID string, limitMonths int) ([]entity.UnitHistory, error) {
	return uc.repo.GetUnitHistory(ctx, tenantSchema, assetID, limitMonths)
}
