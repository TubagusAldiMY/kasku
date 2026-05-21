package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
)

type ListBudgetsUseCase struct {
	repo repository.BudgetRepository
}

func NewListBudgetsUseCase(repo repository.BudgetRepository) *ListBudgetsUseCase {
	return &ListBudgetsUseCase{repo: repo}
}

func (uc *ListBudgetsUseCase) Execute(ctx context.Context, tenantSchema, userID string) ([]entity.BudgetWithProgress, error) {
	budgets, err := uc.repo.List(ctx, tenantSchema, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal list anggaran: %w", err)
	}
	return budgets, nil
}
