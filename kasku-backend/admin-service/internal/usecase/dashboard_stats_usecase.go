package usecase

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
)

// DashboardStatsUseCase mengagregasi statistik untuk halaman dashboard admin (F-ADM-02).
type DashboardStatsUseCase interface {
	Execute(ctx context.Context) (*entity.DashboardStats, error)
}

type dashboardStatsUseCase struct {
	userRepo    repository.UserReadRepository
	paymentRepo repository.PaymentReadRepository
}

// NewDashboardStatsUseCase membuat instance.
func NewDashboardStatsUseCase(userRepo repository.UserReadRepository, paymentRepo repository.PaymentReadRepository) DashboardStatsUseCase {
	return &dashboardStatsUseCase{userRepo: userRepo, paymentRepo: paymentRepo}
}

func (uc *dashboardStatsUseCase) Execute(ctx context.Context) (*entity.DashboardStats, error) {
	stats := &entity.DashboardStats{}
	now := time.Now().UTC()
	since30d := now.AddDate(0, 0, -30)
	since7d := now.AddDate(0, 0, -7)

	totalUsers, err := uc.userRepo.CountTotal(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = totalUsers

	activeUsers, err := uc.userRepo.CountActive(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalActiveUsers = activeUsers

	newLast7d, err := uc.userRepo.CountCreatedSince(ctx, since7d)
	if err != nil {
		return nil, err
	}
	stats.NewUsersLast7Days = newLast7d

	mrr, err := uc.paymentRepo.CountMRRActive(ctx)
	if err != nil {
		return nil, err
	}
	stats.MRRIDR = mrr

	tier, err := uc.paymentRepo.CountByTier(ctx)
	if err != nil {
		return nil, err
	}
	if tier == nil {
		tier = map[string]int64{}
	}
	stats.TierDistribution = tier

	cancelledLast30d, err := uc.paymentRepo.CountCancelledSince(ctx, since30d)
	if err != nil {
		return nil, err
	}
	// Churn ≈ cancelled_30d / total_active (approximation; full cohort analysis di luar scope MVP)
	if activeUsers > 0 {
		stats.ChurnRate30dPct = float64(cancelledLast30d) / float64(activeUsers) * 100
	}

	return stats, nil
}
