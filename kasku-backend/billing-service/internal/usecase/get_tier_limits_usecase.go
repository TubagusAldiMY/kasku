package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
)

// freeTierDefaultLimits adalah fallback limit ketika user belum memiliki subscription.
// Nilai ini sinkron dengan seed data di migration 000002.
var freeTierDefaultLimits = &entity.PlanLimits{
	MaxTransactionsPerMonth:   50,
	MaxFinancialAccounts:      3,
	MaxInvestmentInstruments:  0,
	HistoryRetentionMonths:    3,
	EmailNotificationsEnabled: false,
	ExportCsvEnabled:          false,
}

// GetTierLimitsUseCase mengambil tier limits berdasarkan subscription aktif user.
// Jika user tidak memiliki subscription, dikembalikan FREE tier defaults.
type GetTierLimitsUseCase struct {
	subRepo repository.SubscriptionRepository
}

func NewGetTierLimitsUseCase(subRepo repository.SubscriptionRepository) *GetTierLimitsUseCase {
	return &GetTierLimitsUseCase{subRepo: subRepo}
}

// Execute mengambil PlanLimits untuk userID yang diberikan.
// Dipanggil oleh gRPC handler untuk setiap JWT verification request dari api-gateway.
func (uc *GetTierLimitsUseCase) Execute(ctx context.Context, userID string) (*entity.PlanLimits, error) {
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == domainerrors.ErrSubscriptionNotFound {
			// User belum berlangganan — kembalikan FREE tier limits sebagai default aman
			return freeTierDefaultLimits, nil
		}
		return nil, fmt.Errorf("gagal membaca subscription untuk user %s: %w", userID, err)
	}

	plan, err := uc.subRepo.GetPlanWithLimits(ctx, sub.PlanID.String())
	if err != nil {
		if err == domainerrors.ErrPlanNotFound {
			// Plan tidak aktif lagi — fallback ke FREE tier agar tidak blocking
			return freeTierDefaultLimits, nil
		}
		return nil, fmt.Errorf("gagal membaca plan limits untuk plan %s: %w", sub.PlanID.String(), err)
	}

	return &plan.Limits, nil
}
