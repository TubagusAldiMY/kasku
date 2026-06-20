package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
	"github.com/google/uuid"
)

type CreateDebtInput struct {
	TenantSchema string
	UserID       string
	Direction    entity.DebtDirection
	PersonName   string
	TotalAmount  int64
	DueDate      *time.Time
	Notes        string
}

type CreateDebtUseCase struct {
	repo repository.DebtRepository
}

func NewCreateDebtUseCase(repo repository.DebtRepository) *CreateDebtUseCase {
	return &CreateDebtUseCase{repo: repo}
}

func (uc *CreateDebtUseCase) Execute(ctx context.Context, input CreateDebtInput) (*entity.Debt, error) {
	if input.PersonName == "" {
		return nil, fmt.Errorf("%w: nama orang wajib diisi", domainerrors.ErrInvalidInput)
	}
	if input.TotalAmount <= 0 {
		return nil, fmt.Errorf("%w: nominal hutang harus lebih dari 0", domainerrors.ErrInvalidInput)
	}
	if input.Direction != entity.DebtDirectionReceivable && input.Direction != entity.DebtDirectionPayable {
		return nil, fmt.Errorf("%w: direction tidak valid", domainerrors.ErrInvalidInput)
	}

	now := time.Now().UTC()
	debt := &entity.Debt{
		ID:              uuid.New(),
		UserID:          uuid.MustParse(input.UserID),
		Direction:       input.Direction,
		PersonName:      input.PersonName,
		TotalAmount:     input.TotalAmount,
		RemainingAmount: input.TotalAmount,
		DueDate:         input.DueDate,
		Notes:           input.Notes,
		Status:          entity.DebtStatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := uc.repo.Create(ctx, input.TenantSchema, debt); err != nil {
		return nil, fmt.Errorf("gagal buat catatan hutang: %w", err)
	}
	return debt, nil
}
