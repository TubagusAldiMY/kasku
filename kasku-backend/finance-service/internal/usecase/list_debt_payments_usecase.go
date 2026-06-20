package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
)

type ListDebtPaymentsUseCase struct {
	repo repository.DebtRepository
}

func NewListDebtPaymentsUseCase(repo repository.DebtRepository) *ListDebtPaymentsUseCase {
	return &ListDebtPaymentsUseCase{repo: repo}
}

func (uc *ListDebtPaymentsUseCase) Execute(ctx context.Context, tenantSchema, debtID, userID string) ([]entity.DebtPayment, error) {
	// Verifikasi kepemilikan debt sebelum list payments
	if _, err := uc.repo.GetByID(ctx, tenantSchema, debtID, userID); err != nil {
		return nil, err
	}

	payments, err := uc.repo.ListPayments(ctx, tenantSchema, debtID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil riwayat pembayaran: %w", err)
	}
	if payments == nil {
		payments = []entity.DebtPayment{}
	}
	return payments, nil
}
