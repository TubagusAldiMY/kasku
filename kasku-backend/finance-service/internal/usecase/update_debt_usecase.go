package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type UpdateDebtInput struct {
	TenantSchema string
	ID           string
	UserID       string
	PersonName   string
	DueDate      *time.Time
	Notes        string
}

type UpdateDebtUseCase struct {
	repo repository.DebtRepository
}

func NewUpdateDebtUseCase(repo repository.DebtRepository) *UpdateDebtUseCase {
	return &UpdateDebtUseCase{repo: repo}
}

func (uc *UpdateDebtUseCase) Execute(ctx context.Context, input UpdateDebtInput) (*entity.Debt, error) {
	if input.PersonName == "" {
		return nil, fmt.Errorf("%w: nama orang wajib diisi", domainerrors.ErrInvalidInput)
	}

	debt, err := uc.repo.GetByID(ctx, input.TenantSchema, input.ID, input.UserID)
	if err != nil {
		return nil, err
	}

	debt.PersonName = input.PersonName
	debt.DueDate = input.DueDate
	debt.Notes = input.Notes
	debt.UpdatedAt = time.Now().UTC()

	if err := uc.repo.Update(ctx, input.TenantSchema, debt); err != nil {
		return nil, fmt.Errorf("gagal update hutang: %w", err)
	}
	return debt, nil
}
