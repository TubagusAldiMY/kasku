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

type CreateTransactionInput struct {
	TenantSchema    string
	UserID          string
	SyncID          string
	AccountID       string
	CategoryID      string
	TransactionType entity.TransactionType
	AmountIDR       int64
	TransactionDate time.Time
	Notes           string
	ToAccountID     string
	MaxTransactions int // -1 = unlimited
}

type CreateTransactionUseCase struct {
	txRepo  repository.TransactionRepository
	catRepo repository.CategoryRepository
}

func NewCreateTransactionUseCase(txRepo repository.TransactionRepository, catRepo repository.CategoryRepository) *CreateTransactionUseCase {
	return &CreateTransactionUseCase{txRepo: txRepo, catRepo: catRepo}
}

func (uc *CreateTransactionUseCase) Execute(ctx context.Context, input CreateTransactionInput) (*entity.Transaction, error) {
	if input.AmountIDR <= 0 {
		return nil, fmt.Errorf("%w: amount harus lebih dari 0", domainerrors.ErrInvalidInput)
	}

	// Cek tier limit bulanan jika bukan unlimited
	if input.MaxTransactions >= 0 {
		now := time.Now().UTC()
		count, err := uc.txRepo.CountMonthly(ctx, input.TenantSchema, input.UserID, now)
		if err != nil {
			return nil, fmt.Errorf("gagal cek limit transaksi: %w", err)
		}
		if count >= input.MaxTransactions {
			return nil, domainerrors.ErrTransactionLimitReached
		}
	}

	syncID := input.SyncID
	if syncID == "" {
		syncID = uuid.New().String()
	}

	now := time.Now().UTC()
	tx := &entity.Transaction{
		ID:              uuid.New(),
		SyncID:          syncID,
		TransactionType: input.TransactionType,
		AmountIDR:       input.AmountIDR,
		TransactionDate: input.TransactionDate,
		Notes:           input.Notes,
		IsDeleted:       false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if id, err := uuid.Parse(input.AccountID); err == nil {
		tx.AccountID = id
	}
	if input.CategoryID != "" {
		if id, err := uuid.Parse(input.CategoryID); err == nil {
			tx.CategoryID = &id
		}
	}
	if input.ToAccountID != "" {
		if id, err := uuid.Parse(input.ToAccountID); err == nil {
			tx.ToAccountID = &id
		}
	}

	if err := uc.txRepo.Create(ctx, input.TenantSchema, tx); err != nil {
		return nil, fmt.Errorf("gagal buat transaksi: %w", err)
	}

	return tx, nil
}
