package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type ListDebtsInput struct {
	TenantSchema string
	UserID       string
	Direction    *entity.DebtDirection // nil = semua
}

type ListDebtsUseCase struct {
	repo repository.DebtRepository
}

func NewListDebtsUseCase(repo repository.DebtRepository) *ListDebtsUseCase {
	return &ListDebtsUseCase{repo: repo}
}

func (uc *ListDebtsUseCase) Execute(ctx context.Context, input ListDebtsInput) ([]entity.Debt, error) {
	debts, err := uc.repo.List(ctx, input.TenantSchema, input.UserID, input.Direction)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil daftar hutang: %w", err)
	}
	if debts == nil {
		debts = []entity.Debt{}
	}
	return debts, nil
}
