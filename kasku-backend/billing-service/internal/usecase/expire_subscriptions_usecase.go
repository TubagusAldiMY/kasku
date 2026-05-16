package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/messaging"
	"github.com/rs/zerolog"
)

// ExpireSubscriptionsUseCase menjadi business logic untuk cron expiry.
// Dependensi outbox via repository.ExpireSubscriptionAtomic — usecase tidak
// menyentuh pgx.Tx langsung sehingga tetap framework-agnostic.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_expire_subscriptions_usecase.go -package=mocks
type ExpireSubscriptionsUseCase interface {
	Execute(ctx context.Context) (processed int, err error)
}

type expireSubscriptionsUseCase struct {
	subRepo repository.SubscriptionRepository
	log     zerolog.Logger
}

// NewExpireSubscriptionsUseCase membuat usecase dengan repository terinjeksi.
func NewExpireSubscriptionsUseCase(subRepo repository.SubscriptionRepository, log zerolog.Logger) ExpireSubscriptionsUseCase {
	return &expireSubscriptionsUseCase{subRepo: subRepo, log: log}
}

// Execute list subscription yg sudah lewat period_end, untuk masing-masing flip
// status ke EXPIRED + insert outbox event subscription.expired dalam satu TX.
// Loop melanjutkan walau satu subscription gagal — log warning dan increment metric.
// Mengembalikan jumlah subscription yang berhasil di-flip.
func (uc *expireSubscriptionsUseCase) Execute(ctx context.Context) (int, error) {
	expired, err := uc.subRepo.ListExpiredSubscriptions(ctx)
	if err != nil {
		return 0, fmt.Errorf("gagal list expired subscriptions: %w", err)
	}
	if len(expired) == 0 {
		return 0, nil
	}

	// Cache plan name supaya tidak query berulang untuk plan_id yg sama.
	planNameCache := make(map[string]string)
	processed := 0
	for _, sub := range expired {
		planName, err := uc.resolvePlanName(ctx, sub.PlanID.String(), planNameCache)
		if err != nil {
			// Plan tidak ditemukan — log + skip; sebaiknya tidak gagalkan run.
			uc.log.Warn().
				Err(err).
				Str("subscription_id", sub.ID.String()).
				Str("plan_id", sub.PlanID.String()).
				Msg("gagal resolve plan name, skip expire")
			continue
		}

		event := messaging.SubscriptionExpiredEvent{
			SubscriptionID: sub.ID.String(),
			UserID:         sub.UserID.String(),
			PlanName:       planName,
			PreviousStatus: string(sub.Status),
			ExpiredAt:      time.Now().UTC().Format(time.RFC3339),
		}
		payload, err := json.Marshal(event)
		if err != nil {
			uc.log.Error().Err(err).Str("subscription_id", sub.ID.String()).Msg("gagal marshal expired event")
			continue
		}

		flipped, err := uc.subRepo.ExpireSubscriptionAtomic(
			ctx,
			sub.ID.String(),
			"subscription.expired",
			messaging.RoutingKeySubscriptionExpired,
			payload,
		)
		if err != nil {
			uc.log.Error().
				Err(err).
				Str("subscription_id", sub.ID.String()).
				Msg("gagal expire subscription atomik")
			continue
		}
		if !flipped {
			// Sudah EXPIRED dari run sebelumnya — wajar, skip diam.
			continue
		}
		processed++
	}
	return processed, nil
}

// resolvePlanName meng-cache lookup plan name per plan_id.
func (uc *expireSubscriptionsUseCase) resolvePlanName(ctx context.Context, planID string, cache map[string]string) (string, error) {
	if name, ok := cache[planID]; ok {
		return name, nil
	}
	plan, err := uc.subRepo.GetPlanWithLimits(ctx, planID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrPlanNotFound) {
			return "", err
		}
		return "", fmt.Errorf("gagal lookup plan %s: %w", planID, err)
	}
	cache[planID] = plan.Name
	return plan.Name, nil
}
