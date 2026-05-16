package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
)

// GetUserDetailUseCase mengambil detail satu user + subscription-nya.
type GetUserDetailUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID) (*entity.UserDetail, error)
}

type getUserDetailUseCase struct {
	userRepo repository.UserReadRepository
	subRepo  repository.SubscriptionRepository
}

// NewGetUserDetailUseCase membuat instance.
func NewGetUserDetailUseCase(userRepo repository.UserReadRepository, subRepo repository.SubscriptionRepository) GetUserDetailUseCase {
	return &getUserDetailUseCase{userRepo: userRepo, subRepo: subRepo}
}

func (uc *getUserDetailUseCase) Execute(ctx context.Context, userID uuid.UUID) (*entity.UserDetail, error) {
	summary, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, domainerrors.ErrUserNotFound
	}

	detail := &entity.UserDetail{UserSummary: *summary}

	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub != nil {
		detail.SubscriptionTier = sub.PlanName
		detail.SubscriptionStatus = sub.Status
		detail.SubscriptionID = &sub.ID
		detail.SubscriptionStartedAt = &sub.CurrentPeriodStart
		detail.SubscriptionEndsAt = sub.CurrentPeriodEnd
		detail.SubscriptionPriceIDR = sub.PriceIDR
	} else {
		detail.SubscriptionTier = "FREE"
		detail.SubscriptionStatus = "NONE"
	}

	// Usage stats (TotalTransactions/Accounts/Investments) tidak di-populate di MVP
	// karena admin-service tidak join ke network kasku-internal/kasku-data finance.
	// Kalau dibutuhkan, tambah gRPC client ke finance/transaction/investment-service.
	return detail, nil
}
