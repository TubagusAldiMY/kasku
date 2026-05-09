package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type ListAccountsUseCase struct {
	repo repository.FinancialAccountRepository
}

func NewListAccountsUseCase(repo repository.FinancialAccountRepository) *ListAccountsUseCase {
	return &ListAccountsUseCase{repo: repo}
}

func (uc *ListAccountsUseCase) Execute(ctx context.Context, tenantSchema, userID string) ([]entity.FinancialAccount, error) {
	accounts, err := uc.repo.List(ctx, tenantSchema, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil daftar akun: %w", err)
	}
	return accounts, nil
}
