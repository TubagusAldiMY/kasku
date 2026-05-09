package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/user-service/internal/infrastructure/persistence"
	"github.com/rs/zerolog"
)

// ProvisionTenantUseCase menangani provisioning tenant baru setelah user registrasi.
type ProvisionTenantUseCase struct {
	financeRepo persistence.FinanceRepository
	billingRepo persistence.BillingRepository
	log         zerolog.Logger
}

func NewProvisionTenantUseCase(
	financeRepo persistence.FinanceRepository,
	billingRepo persistence.BillingRepository,
	log zerolog.Logger,
) *ProvisionTenantUseCase {
	return &ProvisionTenantUseCase{
		financeRepo: financeRepo,
		billingRepo: billingRepo,
		log:         log,
	}
}

func (uc *ProvisionTenantUseCase) Execute(ctx context.Context, userID string) error {
	uc.log.Info().Str("user_id", userID).Msg("memulai provisioning tenant")

	// 1. Provision tenant schema di kasku_finance
	if err := uc.financeRepo.ProvisionTenant(ctx, userID); err != nil {
		return fmt.Errorf("gagal provision tenant: %w", err)
	}

	// 2. Buat subscription FREE di kasku_billing
	if err := uc.billingRepo.CreateFreeSubscription(ctx, userID); err != nil {
		return fmt.Errorf("gagal buat subscription FREE: %w", err)
	}

	uc.log.Info().Str("user_id", userID).Msg("provisioning tenant selesai")
	return nil
}
