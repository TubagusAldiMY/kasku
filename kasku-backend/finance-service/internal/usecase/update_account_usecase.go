package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type UpdateAccountInput struct {
	TenantSchema string
	ID           string
	UserID       string
	Name         string
	AccountType  entity.AccountType
	Color        string
	Icon         string
	IsDefault    bool
}

type UpdateAccountUseCase struct {
	repo repository.FinancialAccountRepository
}

func NewUpdateAccountUseCase(repo repository.FinancialAccountRepository) *UpdateAccountUseCase {
	return &UpdateAccountUseCase{repo: repo}
}

func (uc *UpdateAccountUseCase) Execute(ctx context.Context, input UpdateAccountInput) (*entity.FinancialAccount, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: nama akun wajib diisi", domainerrors.ErrInvalidInput)
	}

	existing, err := uc.repo.GetByID(ctx, input.TenantSchema, input.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil akun: %w", err)
	}

	existing.Name = input.Name
	existing.AccountType = input.AccountType
	existing.Color = input.Color
	existing.Icon = input.Icon
	existing.IsDefault = input.IsDefault

	if err := uc.repo.Update(ctx, input.TenantSchema, existing); err != nil {
		return nil, fmt.Errorf("gagal update akun: %w", err)
	}

	return existing, nil
}
