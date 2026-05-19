package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/TubagusAldiMY/kasku/user-service/internal/infrastructure/persistence"
	"github.com/rs/zerolog"
)

// ProvisionTenantUseCase menangani provisioning tenant baru setelah user registrasi.
type ProvisionTenantUseCase struct {
	financeRepo      persistence.FinanceRepository
	subscriptionRepo persistence.SubscriptionRepository
	profileRepo      persistence.UserProfileRepository
	log              zerolog.Logger
}

func NewProvisionTenantUseCase(
	financeRepo persistence.FinanceRepository,
	subscriptionRepo persistence.SubscriptionRepository,
	profileRepo persistence.UserProfileRepository,
	log zerolog.Logger,
) *ProvisionTenantUseCase {
	return &ProvisionTenantUseCase{
		financeRepo:      financeRepo,
		subscriptionRepo: subscriptionRepo,
		profileRepo:      profileRepo,
		log:              log,
	}
}

func (uc *ProvisionTenantUseCase) Execute(ctx context.Context, userID, email, username string) error {
	uc.log.Info().Str("user_id", userID).Msg("memulai provisioning tenant")

	// 1. Provision tenant schema di kasku_finance
	if err := uc.financeRepo.ProvisionTenant(ctx, userID); err != nil {
		return fmt.Errorf("gagal provision tenant: %w", err)
	}

	tenantSchema := "tenant_" + strings.ReplaceAll(userID, "-", "_")
	if err := uc.financeRepo.EnsureTenantRuntimeObjects(ctx, tenantSchema); err != nil {
		return fmt.Errorf("gagal ensure tenant runtime objects: %w", err)
	}
	if err := uc.financeRepo.RemoveDefaultCategorySeeds(ctx, tenantSchema); err != nil {
		return fmt.Errorf("gagal hapus seed kategori default: %w", err)
	}

	// 2. Buat subscription FREE di kasku_billing
	if err := uc.subscriptionRepo.CreateFreeSubscription(ctx, userID); err != nil {
		return fmt.Errorf("gagal buat subscription FREE: %w", err)
	}

	// 3. Buat user profile di kasku_user
	if err := uc.profileRepo.EnsureUserProfile(ctx, userID, email, username); err != nil {
		return fmt.Errorf("gagal buat user profile: %w", err)
	}

	uc.log.Info().Str("user_id", userID).Msg("provisioning tenant selesai")
	return nil
}
