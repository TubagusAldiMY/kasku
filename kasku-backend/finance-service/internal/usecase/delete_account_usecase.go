package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type DeleteAccountUseCase struct {
	repo repository.FinancialAccountRepository
}

func NewDeleteAccountUseCase(repo repository.FinancialAccountRepository) *DeleteAccountUseCase {
	return &DeleteAccountUseCase{repo: repo}
}

func (uc *DeleteAccountUseCase) Execute(ctx context.Context, tenantSchema, id, userID string) error {
	if err := uc.repo.SoftDelete(ctx, tenantSchema, id, userID); err != nil {
		return fmt.Errorf("gagal hapus akun: %w", err)
	}
	return nil
}
