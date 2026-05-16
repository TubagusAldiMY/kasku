package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
)

// ListPaymentsOutput membawa data + total count.
type ListPaymentsOutput struct {
	Payments []entity.PaymentSummary
	Total    int64
}

// ListPaymentsUseCase mengambil daftar payment dari kasku_billing.
type ListPaymentsUseCase interface {
	Execute(ctx context.Context, filter repository.PaymentListFilter) (*ListPaymentsOutput, error)
}

type listPaymentsUseCase struct {
	repo repository.PaymentReadRepository
}

// NewListPaymentsUseCase membuat instance.
func NewListPaymentsUseCase(repo repository.PaymentReadRepository) ListPaymentsUseCase {
	return &listPaymentsUseCase{repo: repo}
}

func (uc *listPaymentsUseCase) Execute(ctx context.Context, filter repository.PaymentListFilter) (*ListPaymentsOutput, error) {
	items, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &ListPaymentsOutput{Payments: items, Total: total}, nil
}
