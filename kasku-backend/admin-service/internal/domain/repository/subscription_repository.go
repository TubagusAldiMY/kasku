package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SubscriptionView adalah read-model subscription user yang dipakai admin.
type SubscriptionView struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	PlanID             uuid.UUID
	PlanName           string
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   *time.Time
	PriceIDR           int64
}

// SubscriptionRepository adalah port untuk subscriptions + subscription_plans di kasku_billing.
// Mendukung read (untuk list/detail) dan write (override plan).
type SubscriptionRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*SubscriptionView, error)
	// GetByUserIDs mengembalikan map user_id -> SubscriptionView untuk in-memory join saat list users.
	GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]SubscriptionView, error)
	// FindPlanByName mengembalikan ID + harga plan yang akan diset saat override.
	FindPlanByName(ctx context.Context, name string) (planID uuid.UUID, priceIDR int64, err error)
	// UpdatePlan mengubah plan_id + current_period_start (current_period_end di-reset ke null sampai pembayaran berikutnya).
	UpdatePlan(ctx context.Context, subscriptionID uuid.UUID, newPlanID uuid.UUID, periodStart time.Time) error
}
