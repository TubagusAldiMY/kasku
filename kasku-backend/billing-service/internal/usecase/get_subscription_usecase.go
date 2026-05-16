package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
)

// GetSubscriptionUseCase mengambil detail subscription milik user.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_get_subscription_usecase.go -package=mocks
type GetSubscriptionUseCase interface {
	Execute(ctx context.Context, userID string) (*entity.Subscription, error)
}

type getSubscriptionUseCase struct {
	subRepo repository.SubscriptionRepository
}

func NewGetSubscriptionUseCase(subRepo repository.SubscriptionRepository) GetSubscriptionUseCase {
	return &getSubscriptionUseCase{subRepo: subRepo}
}

// Execute mengembalikan subscription aktif user — termasuk status dan periode aktif.
// Caller bertanggung jawab memeriksa ErrSubscriptionNotFound dari domain errors.
func (uc *getSubscriptionUseCase) Execute(ctx context.Context, userID string) (*entity.Subscription, error) {
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil subscription untuk user %s: %w", userID, err)
	}
	return sub, nil
}
