package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
)

type ListTransactionsInput struct {
	TenantSchema  string
	UserID        string
	From          time.Time
	To            time.Time
	HistoryMonths int // -1 = unlimited, 0 = default (current month)
}

type ListTransactionsResult struct {
	Transactions []entity.Transaction
	Summary      *entity.TransactionSummary
}

type ListTransactionsUseCase struct {
	txRepo repository.TransactionRepository
}

func NewListTransactionsUseCase(txRepo repository.TransactionRepository) *ListTransactionsUseCase {
	return &ListTransactionsUseCase{txRepo: txRepo}
}

func (uc *ListTransactionsUseCase) Execute(ctx context.Context, input ListTransactionsInput) (*ListTransactionsResult, error) {
	from := input.From
	to := input.To

	// Default: bulan berjalan jika tidak ada range
	if from.IsZero() {
		now := time.Now().UTC()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to = from.AddDate(0, 1, 0).Add(-time.Second)
	}

	// Enforce history retention limit berdasarkan tier subscription
	if input.HistoryMonths > 0 {
		earliest := time.Now().UTC().AddDate(0, -input.HistoryMonths, 0)
		if from.Before(earliest) {
			from = earliest
		}
	}

	txs, err := uc.txRepo.List(ctx, input.TenantSchema, input.UserID, from, to)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil daftar transaksi: %w", err)
	}

	summary, err := uc.txRepo.GetSummary(ctx, input.TenantSchema, input.UserID, from, to)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil summary: %w", err)
	}

	return &ListTransactionsResult{
		Transactions: txs,
		Summary:      summary,
	}, nil
}
