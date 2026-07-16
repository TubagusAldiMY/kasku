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

	debtID, err := uuid.Parse(input.DebtID)
	if err != nil {
		return nil, fmt.Errorf("%w: debt id tidak valid", domainerrors.ErrInvalidInput)
	}

	payment := &entity.DebtPayment{
		ID:          uuid.New(),
		DebtID:      debtID,
		Amount:      input.Amount,
		PaymentDate: input.PaymentDate,
		Notes:       input.Notes,
		CreatedAt:   time.Now().UTC(),
	}

	// RecordPayment melakukan insert payment + deduct remaining secara atomik
	// (SELECT ... FOR UPDATE). Validasi status/jumlah dilakukan di dalam transaksi
	// terhadap nilai terkunci sehingga tidak ada race check-then-act.
	if err := uc.repo.RecordPayment(ctx, input.TenantSchema, input.DebtID, input.UserID, payment); err != nil {
		return nil, err
	}

	return payment, nil
}
