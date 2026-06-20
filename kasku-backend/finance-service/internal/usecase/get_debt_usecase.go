package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type GetDebtUseCase struct {
	repo repository.DebtRepository
}

func NewGetDebtUseCase(repo repository.DebtRepository) *GetDebtUseCase {
	return &GetDebtUseCase{repo: repo}
}

func (uc *GetDebtUseCase) Execute(ctx context.Context, tenantSchema, id, userID string) (*entity.Debt, error) {
	debt, err := uc.repo.GetByID(ctx, tenantSchema, id, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil hutang: %w", err)
	}
	return debt, nil
}
