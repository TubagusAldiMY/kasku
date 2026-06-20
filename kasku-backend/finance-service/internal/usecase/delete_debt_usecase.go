package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type DeleteDebtUseCase struct {
	repo repository.DebtRepository
}

func NewDeleteDebtUseCase(repo repository.DebtRepository) *DeleteDebtUseCase {
	return &DeleteDebtUseCase{repo: repo}
}

func (uc *DeleteDebtUseCase) Execute(ctx context.Context, tenantSchema, id, userID string) error {
	if err := uc.repo.Delete(ctx, tenantSchema, id, userID); err != nil {
		return fmt.Errorf("gagal hapus hutang: %w", err)
	}
	return nil
}
