package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
)

type DeleteTransactionUseCase struct {
	txRepo repository.TransactionRepository
}

func NewDeleteTransactionUseCase(txRepo repository.TransactionRepository) *DeleteTransactionUseCase {
	return &DeleteTransactionUseCase{txRepo: txRepo}
}

func (uc *DeleteTransactionUseCase) Execute(ctx context.Context, tenantSchema, id, userID string) error {
	if err := uc.txRepo.SoftDelete(ctx, tenantSchema, id, userID); err != nil {
		return fmt.Errorf("gagal hapus transaksi: %w", err)
	}
	return nil
}
