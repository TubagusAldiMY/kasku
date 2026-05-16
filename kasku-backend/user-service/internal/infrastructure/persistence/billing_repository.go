package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SubscriptionRepository mendefinisikan operasi ke kasku_billing.subscriptions
// yang dipanggil oleh user-service saat provisioning user baru.
//
// Tabel subscriptions dimiliki billing-service; user-service hanya boleh
// INSERT baris FREE saat onboarding lewat event user.registered.
// Cross-DB grant terdokumentasi di infra/postgres/00-init-databases.sh.
type SubscriptionRepository interface {
	CreateFreeSubscription(ctx context.Context, userID string) error
}

type postgresSubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSubscriptionRepository(pool *pgxpool.Pool) SubscriptionRepository {
	return &postgresSubscriptionRepository{pool: pool}
}

// CreateFreeSubscription membuat subscription FREE untuk user baru.
// ON CONFLICT DO NOTHING menjamin idempotency.
func (r *postgresSubscriptionRepository) CreateFreeSubscription(ctx context.Context, userID string) error {
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
