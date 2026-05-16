package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/repository"
)

type PriceProvider interface {
	GetPrice(ctx context.Context, symbol string) (priceIDR float64, isFresh bool, err error)
}

type ListAssetsUseCase struct {
	repo          repository.InvestmentAssetRepository
	priceProvider PriceProvider
}

func NewListAssetsUseCase(repo repository.InvestmentAssetRepository, priceProvider PriceProvider) *ListAssetsUseCase {
	return &ListAssetsUseCase{repo: repo, priceProvider: priceProvider}
}

func (uc *ListAssetsUseCase) Execute(ctx context.Context, tenantSchema string) ([]entity.InvestmentAsset, error) {
	assets, err := uc.repo.List(ctx, tenantSchema)
	if err != nil {
		return nil, err
	}

	if uc.priceProvider == nil {
		return assets, nil
	}

	for i := range assets {
		priceIDR, isFresh, err := uc.priceProvider.GetPrice(ctx, assets[i].Symbol)
		if err != nil {
			continue
		}
		assets[i].CurrentPrice = &priceIDR
		assets[i].IsPriceFresh = &isFresh
	}

	return assets, nil
}
