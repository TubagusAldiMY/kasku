package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

const unlimitedHistoryMonths = 0 // 0 = semua history tanpa batasan bulan

type GetBalanceHistoryUseCase struct {
	repo repository.FinancialAccountRepository
}

func NewGetBalanceHistoryUseCase(repo repository.FinancialAccountRepository) *GetBalanceHistoryUseCase {
	return &GetBalanceHistoryUseCase{repo: repo}
}

func (uc *GetBalanceHistoryUseCase) Execute(ctx context.Context, tenantSchema, accountID, userID string, historyMonths int) ([]entity.BalanceHistory, error) {
	// Verifikasi kepemilikan akun sebelum mengembalikan history
	if _, err := uc.repo.GetByID(ctx, tenantSchema, accountID, userID); err != nil {
		return nil, fmt.Errorf("akun tidak ditemukan: %w", err)
	}

	limitMonths := historyMonths
	if limitMonths < 0 {
		limitMonths = unlimitedHistoryMonths
	}

	history, err := uc.repo.GetBalanceHistory(ctx, tenantSchema, accountID, limitMonths)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil balance history: %w", err)
	}
	return history, nil
}
