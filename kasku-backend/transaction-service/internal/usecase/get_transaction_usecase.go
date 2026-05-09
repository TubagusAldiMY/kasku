package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
)

type GetTransactionUseCase struct {
	txRepo repository.TransactionRepository
}

func NewGetTransactionUseCase(txRepo repository.TransactionRepository) *GetTransactionUseCase {
	return &GetTransactionUseCase{txRepo: txRepo}
}

func (uc *GetTransactionUseCase) Execute(ctx context.Context, tenantSchema, id, userID string) (*entity.Transaction, error) {
	tx, err := uc.txRepo.GetByID(ctx, tenantSchema, id, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil transaksi: %w", err)
	}
	return tx, nil
}
