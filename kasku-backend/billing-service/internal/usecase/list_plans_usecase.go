package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
)

// ListPlansUseCase mengambil semua subscription plan yang aktif.
type ListPlansUseCase struct {
	subRepo repository.SubscriptionRepository
}

func NewListPlansUseCase(subRepo repository.SubscriptionRepository) *ListPlansUseCase {
	return &ListPlansUseCase{subRepo: subRepo}
}

// Execute mengembalikan daftar plan yang tersedia untuk ditampilkan di halaman pricing.
func (uc *ListPlansUseCase) Execute(ctx context.Context) ([]entity.SubscriptionPlan, error) {
	plans, err := uc.subRepo.ListAllPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar subscription plan: %w", err)
	}
	return plans, nil
}
