package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
)

// ListUsersInput hampir sama dengan repository.UserListFilter,
// tetapi juga menerima filter `tier` yang di-resolve secara in-memory
// setelah merge dengan kasku_billing.subscriptions.
type ListUsersInput struct {
	Filter repository.UserListFilter
	Tier   *string // opsional — filter by subscription_plans.name
}

// ListUsersOutput membawa data + total count untuk pagination meta.
type ListUsersOutput struct {
	Users []entity.UserSummary
	Total int64
}

// ListUsersUseCase mengambil daftar user + merge tier dari billing.
type ListUsersUseCase interface {
	Execute(ctx context.Context, in ListUsersInput) (*ListUsersOutput, error)
}

type listUsersUseCase struct {
	userRepo repository.UserReadRepository
	subRepo  repository.SubscriptionRepository
}

// NewListUsersUseCase membuat instance.
func NewListUsersUseCase(userRepo repository.UserReadRepository, subRepo repository.SubscriptionRepository) ListUsersUseCase {
	return &listUsersUseCase{userRepo: userRepo, subRepo: subRepo}
}

func (uc *listUsersUseCase) Execute(ctx context.Context, in ListUsersInput) (*ListUsersOutput, error) {
	users, total, err := uc.userRepo.List(ctx, in.Filter)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return &ListUsersOutput{Users: users, Total: total}, nil
	}

	ids := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	subs, err := uc.subRepo.GetByUserIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	merged := make([]entity.UserSummary, 0, len(users))
	for _, u := range users {
		if s, ok := subs[u.ID]; ok {
			u.SubscriptionTier = s.PlanName
			u.SubscriptionStatus = s.Status
		} else {
			u.SubscriptionTier = "FREE"
			u.SubscriptionStatus = "NONE"
		}
		if in.Tier != nil && u.SubscriptionTier != *in.Tier {
			continue
		}
		merged = append(merged, u)
	}

	// Catatan: total tetap pakai count pre-filter tier karena tier filter di-resolve in-memory.
	// Untuk UI yang konsisten, frontend mestinya nge-filter tier via dedicated query yang JOIN cross-DB —
	// di MVP cukup tampilkan total = filtered length kalau tier di-set.
	finalTotal := total
	if in.Tier != nil {
		finalTotal = int64(len(merged))
	}

	return &ListUsersOutput{Users: merged, Total: finalTotal}, nil
}
