package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
)

type DeleteBudgetUseCase struct {
	repo repository.BudgetRepository
}

func NewDeleteBudgetUseCase(repo repository.BudgetRepository) *DeleteBudgetUseCase {
	return &DeleteBudgetUseCase{repo: repo}
}

func (uc *DeleteBudgetUseCase) Execute(ctx context.Context, tenantSchema, id, userID string) error {
	if err := uc.repo.SoftDelete(ctx, tenantSchema, id, userID); err != nil {
		return fmt.Errorf("gagal hapus anggaran: %w", err)
	}
	return nil
}
