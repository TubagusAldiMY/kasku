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

type RecordDebtPaymentInput struct {
	TenantSchema string
	DebtID       string
	UserID       string
	Amount       int64
	PaymentDate  time.Time
	Notes        string
}

type RecordDebtPaymentUseCase struct {
	repo repository.DebtRepository
}

func NewRecordDebtPaymentUseCase(repo repository.DebtRepository) *RecordDebtPaymentUseCase {
	return &RecordDebtPaymentUseCase{repo: repo}
}

func (uc *RecordDebtPaymentUseCase) Execute(ctx context.Context, input RecordDebtPaymentInput) (*entity.DebtPayment, error) {
	if input.Amount <= 0 {
		return nil, fmt.Errorf("%w: jumlah pembayaran harus lebih dari 0", domainerrors.ErrInvalidInput)
	}

	debt, err := uc.repo.GetByID(ctx, input.TenantSchema, input.DebtID, input.UserID)
	if err != nil {
		return nil, err
	}

	if debt.Status == entity.DebtStatusSettled {
		return nil, domainerrors.ErrDebtAlreadySettled
	}
	if input.Amount > debt.RemainingAmount {
		return nil, domainerrors.ErrPaymentExceedsDebt
	}

	payment := &entity.DebtPayment{
		ID:          uuid.New(),
		DebtID:      debt.ID,
		Amount:      input.Amount,
		PaymentDate: input.PaymentDate,
		Notes:       input.Notes,
		CreatedAt:   time.Now().UTC(),
	}

	if err := uc.repo.CreatePayment(ctx, input.TenantSchema, payment); err != nil {
		return nil, fmt.Errorf("gagal catat pembayaran: %w", err)
	}
	if err := uc.repo.DeductRemaining(ctx, input.TenantSchema, input.DebtID, input.Amount); err != nil {
		return nil, fmt.Errorf("gagal update sisa hutang: %w", err)
	}

	return payment, nil
}
