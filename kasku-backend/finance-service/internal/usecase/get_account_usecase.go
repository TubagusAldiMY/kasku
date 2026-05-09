package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type GetAccountUseCase struct {
	repo repository.FinancialAccountRepository
}

func NewGetAccountUseCase(repo repository.FinancialAccountRepository) *GetAccountUseCase {
	return &GetAccountUseCase{repo: repo}
}

func (uc *GetAccountUseCase) Execute(ctx context.Context, tenantSchema, id, userID string) (*entity.FinancialAccount, error) {
	account, err := uc.repo.GetByID(ctx, tenantSchema, id, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil akun: %w", err)
	}
	return account, nil
}
