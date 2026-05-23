package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
	"github.com/google/uuid"
)

type UpdateTransactionInput struct {
	TenantSchema    string
	UserID          string
	ID              string
	AccountID       string
	CategoryID      string
	BudgetID        string
	TransactionType entity.TransactionType
	AmountIDR       int64
	TransactionDate time.Time
	Notes           string
	ToAccountID     string
}

type UpdateTransactionUseCase struct {
	txRepo repository.TransactionRepository
}

func NewUpdateTransactionUseCase(txRepo repository.TransactionRepository) *UpdateTransactionUseCase {
	return &UpdateTransactionUseCase{txRepo: txRepo}
}

func (uc *UpdateTransactionUseCase) Execute(ctx context.Context, input UpdateTransactionInput) (*entity.Transaction, error) {
	if input.AmountIDR <= 0 {
		return nil, fmt.Errorf("%w: amount harus lebih dari 0", domainerrors.ErrInvalidInput)
	}
	if input.TransactionType != entity.TransactionIncome &&
		input.TransactionType != entity.TransactionExpense &&
		input.TransactionType != entity.TransactionTransfer {
		return nil, fmt.Errorf("%w: tipe transaksi tidak valid", domainerrors.ErrInvalidInput)
	}
	if input.TransactionType == entity.TransactionTransfer {
		if input.ToAccountID == "" || input.ToAccountID == input.AccountID {
			return nil, fmt.Errorf("%w: rekening tujuan transfer tidak valid", domainerrors.ErrInvalidInput)
		}
	}

	txID, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: ID transaksi tidak valid", domainerrors.ErrInvalidInput)
	}
	accountID, err := uuid.Parse(input.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: ID rekening tidak valid", domainerrors.ErrInvalidInput)
	}

	tx := &entity.Transaction{
		ID:              txID,
		AccountID:       accountID,
		TransactionType: input.TransactionType,
		AmountIDR:       input.AmountIDR,
		TransactionDate: input.TransactionDate,
		Notes:           input.Notes,
	}

	if input.CategoryID != "" {
		id, err := uuid.Parse(input.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("%w: ID kategori tidak valid", domainerrors.ErrInvalidInput)
		}
		tx.CategoryID = &id
	}
	if input.ToAccountID != "" {
		id, err := uuid.Parse(input.ToAccountID)
		if err != nil {
			return nil, fmt.Errorf("%w: ID rekening tujuan tidak valid", domainerrors.ErrInvalidInput)
		}
		tx.ToAccountID = &id
	}
	if input.TransactionType == entity.TransactionExpense && input.BudgetID != "" {
		id, err := uuid.Parse(input.BudgetID)
		if err != nil {
			return nil, fmt.Errorf("%w: ID anggaran tidak valid", domainerrors.ErrInvalidInput)
		}
		tx.BudgetID = &id
	}

	if err := uc.txRepo.Update(ctx, input.TenantSchema, input.UserID, tx); err != nil {
		return nil, fmt.Errorf("gagal update transaksi: %w", err)
	}

	return tx, nil
}
