package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
)

type GetBudgetUseCase struct {
	repo repository.BudgetRepository
}

func NewGetBudgetUseCase(repo repository.BudgetRepository) *GetBudgetUseCase {
	return &GetBudgetUseCase{repo: repo}
}

func (uc *GetBudgetUseCase) Execute(ctx context.Context, tenantSchema, id, userID string) (*entity.BudgetWithProgress, error) {
	b, err := uc.repo.GetByID(ctx, tenantSchema, id, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal get anggaran: %w", err)
	}
	return b, nil
}
