package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresSubscriptionRepository punya akses R/W ke kasku_billing.subscriptions + subscription_plans.
type postgresSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSubscriptionRepository membuat repository subscription untuk admin-service.
func NewPostgresSubscriptionRepository(pool *pgxpool.Pool) repository.SubscriptionRepository {
	return &postgresSubscriptionRepository{pool: pool}
}

func (r *postgresSubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*repository.SubscriptionView, error) {
	const q = `
		SELECT s.id, s.user_id, s.plan_id, plan.name, s.status,
		       s.current_period_start, s.current_period_end, plan.price_idr
		FROM public.subscriptions s
		JOIN public.subscription_plans plan ON plan.id = s.plan_id
		WHERE s.user_id = $1
		LIMIT 1
	`
	var v repository.SubscriptionView
	err := r.pool.QueryRow(ctx, q, userID).Scan(
		&v.ID, &v.UserID, &v.PlanID, &v.PlanName, &v.Status,
		&v.CurrentPeriodStart, &v.CurrentPeriodEnd, &v.PriceIDR,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal get subscription user: %w", err)
	}
	return &v, nil
}

func (r *postgresSubscriptionRepository) GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]repository.SubscriptionView, error) {
	if len(userIDs) == 0 {
		return map[uuid.UUID]repository.SubscriptionView{}, nil
	}
	const q = `
		SELECT s.id, s.user_id, s.plan_id, plan.name, s.status,
		       s.current_period_start, s.current_period_end, plan.price_idr
		FROM public.subscriptions s
		JOIN public.subscription_plans plan ON plan.id = s.plan_id
		WHERE s.user_id = ANY($1::uuid[])
	`
	rows, err := r.pool.Query(ctx, q, userIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal query subscriptions: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]repository.SubscriptionView, len(userIDs))
	for rows.Next() {
		var v repository.SubscriptionView
		if err := rows.Scan(
			&v.ID, &v.UserID, &v.PlanID, &v.PlanName, &v.Status,
			&v.CurrentPeriodStart, &v.CurrentPeriodEnd, &v.PriceIDR,
		); err != nil {
			return nil, fmt.Errorf("gagal scan subscription: %w", err)
		}
		out[v.UserID] = v
	}
	return out, rows.Err()
}

func (r *postgresSubscriptionRepository) FindPlanByName(ctx context.Context, name string) (uuid.UUID, int64, error) {
	const q = `SELECT id, price_idr FROM public.subscription_plans WHERE UPPER(name) = UPPER($1) AND is_active = true LIMIT 1`
	var (
		id    uuid.UUID
		price int64
	)
	err := r.pool.QueryRow(ctx, q, name).Scan(&id, &price)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, 0, nil
		}
		return uuid.Nil, 0, fmt.Errorf("gagal find plan: %w", err)
	}
	return id, price, nil
}

func (r *postgresSubscriptionRepository) UpdatePlan(ctx context.Context, subID uuid.UUID, newPlanID uuid.UUID, periodStart time.Time) error {
	const q = `
		UPDATE public.subscriptions
		SET plan_id = $2,
		    status = 'ACTIVE',
		    current_period_start = $3,
		    current_period_end = NULL
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, q, subID, newPlanID, periodStart)
	if err != nil {
		return fmt.Errorf("gagal update plan subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("subscription tidak ditemukan saat update plan")
	}
	return nil
}
