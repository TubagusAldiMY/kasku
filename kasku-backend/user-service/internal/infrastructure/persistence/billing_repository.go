package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BillingRepository mendefinisikan operasi ke kasku_billing.
type BillingRepository interface {
	CreateFreeSubscription(ctx context.Context, userID string) error
}

type postgresBillingRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBillingRepository(pool *pgxpool.Pool) BillingRepository {
	return &postgresBillingRepository{pool: pool}
}

// CreateFreeSubscription membuat subscription FREE untuk user baru.
// ON CONFLICT DO NOTHING menjamin idempotency.
func (r *postgresBillingRepository) CreateFreeSubscription(ctx context.Context, userID string) error {
	query := `
		INSERT INTO public.subscriptions (user_id, plan_id, status, current_period_start)
		SELECT $1::uuid, id, 'ACTIVE', now()
		FROM public.subscription_plans
		WHERE name = 'FREE' AND is_active = true
		LIMIT 1
		ON CONFLICT (user_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("gagal insert subscription FREE untuk user %s: %w", userID, err)
	}
	return nil
}
