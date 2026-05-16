package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresSubscriptionRepository mengimplementasikan repository.SubscriptionRepository
// menggunakan pgx/v5 connection pool ke database kasku_billing.
type postgresSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSubscriptionRepository membuat instance repository baru.
func NewPostgresSubscriptionRepository(pool *pgxpool.Pool) repository.SubscriptionRepository {
	return &postgresSubscriptionRepository{pool: pool}
}

// GetByUserID mengambil subscription aktif milik user dari database.
// Mengembalikan ErrSubscriptionNotFound jika user tidak memiliki subscription.
func (r *postgresSubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*entity.Subscription, error) {
	const query = `
		SELECT id, user_id, plan_id, status, current_period_start, current_period_end, created_at, updated_at
		FROM public.subscriptions
		WHERE user_id = $1
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, userID)
	sub := &entity.Subscription{}
	err := row.Scan(
		&sub.ID, &sub.UserID, &sub.PlanID,
		&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("gagal scan baris subscription: %w", err)
	}
	return sub, nil
}

// GetPlanWithLimits mengambil subscription plan beserta tier limitsnya.
// Hanya mengembalikan plan yang is_active = true.
// Mengembalikan ErrPlanNotFound jika plan tidak ditemukan atau tidak aktif.
func (r *postgresSubscriptionRepository) GetPlanWithLimits(ctx context.Context, planID string) (*entity.SubscriptionPlan, error) {
	const query = `
		SELECT id, name, price_idr, limits, is_active
		FROM public.subscription_plans
		WHERE id = $1 AND is_active = true
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, planID)
	plan := &entity.SubscriptionPlan{}
	var limitsJSON []byte
	err := row.Scan(&plan.ID, &plan.Name, &plan.PriceIDR, &limitsJSON, &plan.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrPlanNotFound
		}
		return nil, fmt.Errorf("gagal scan baris subscription plan: %w", err)
	}
	if err := json.Unmarshal(limitsJSON, &plan.Limits); err != nil {
		return nil, fmt.Errorf("gagal unmarshal limits JSON untuk plan %s: %w", planID, err)
	}
	return plan, nil
}

// ListAllPlans mengembalikan semua plan yang aktif, diurutkan dari harga termurah.
func (r *postgresSubscriptionRepository) ListAllPlans(ctx context.Context) ([]entity.SubscriptionPlan, error) {
	const query = `
		SELECT id, name, price_idr, limits, is_active
		FROM public.subscription_plans
		WHERE is_active = true
		ORDER BY price_idr ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query daftar subscription plan: %w", err)
	}
	defer rows.Close()

	var plans []entity.SubscriptionPlan
	for rows.Next() {
		plan := entity.SubscriptionPlan{}
		var limitsJSON []byte
		if err := rows.Scan(&plan.ID, &plan.Name, &plan.PriceIDR, &limitsJSON, &plan.IsActive); err != nil {
			return nil, fmt.Errorf("gagal scan baris plan: %w", err)
		}
		if err := json.Unmarshal(limitsJSON, &plan.Limits); err != nil {
			return nil, fmt.Errorf("gagal unmarshal limits JSON: %w", err)
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi rows plans gagal: %w", err)
	}
	return plans, nil
}

// ListExpiredSubscriptions mengembalikan semua subscription ACTIVE yang sudah melewati period end.
// Digunakan oleh background job untuk update status ke EXPIRED.
func (r *postgresSubscriptionRepository) ListExpiredSubscriptions(ctx context.Context) ([]entity.Subscription, error) {
	const query = `
		SELECT id, user_id, plan_id, status, current_period_start, current_period_end, created_at, updated_at
		FROM public.subscriptions
		WHERE status = 'ACTIVE'
		  AND current_period_end IS NOT NULL
		  AND current_period_end < now()
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query expired subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []entity.Subscription
	for rows.Next() {
		sub := entity.Subscription{}
		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID,
			&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan baris subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi rows expired subscriptions gagal: %w", err)
	}
	return subs, nil
}

// UpdateStatus mengubah status subscription berdasarkan subscriptionID.
// updated_at diperbarui secara otomatis oleh database trigger.
func (r *postgresSubscriptionRepository) UpdateStatus(ctx context.Context, subscriptionID string, status entity.SubscriptionStatus) error {
	const query = `UPDATE public.subscriptions SET status = $2, updated_at = now() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, subscriptionID, string(status))
	if err != nil {
		return fmt.Errorf("gagal update status subscription %s: %w", subscriptionID, err)
	}
	return nil
}

// ExpireSubscriptionAtomic membungkus UPDATE status + INSERT outbox event dalam
// satu transaksi pgx. Guard `AND status = 'ACTIVE'` memastikan operasi idempotent
// kalau cron jalan ulang setelah crash sebelum publish — baris yang sudah EXPIRED
// tidak menghasilkan event duplikat.
func (r *postgresSubscriptionRepository) ExpireSubscriptionAtomic(
	ctx context.Context,
	subscriptionID string,
	eventType string,
	routingKey string,
	payload []byte,
) (bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("gagal mulai transaksi expire subscription: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE public.subscriptions
		SET status = 'EXPIRED', updated_at = now()
		WHERE id = $1 AND status = 'ACTIVE'
	`, subscriptionID)
	if err != nil {
		return false, fmt.Errorf("gagal update status subscription %s: %w", subscriptionID, err)
	}
	if tag.RowsAffected() == 0 {
		// Tidak ada baris ACTIVE — sudah di-flip oleh run sebelumnya.
		return false, nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO public.outbox_events (event_type, routing_key, payload)
		VALUES ($1, $2, $3::jsonb)
	`, eventType, routingKey, string(payload)); err != nil {
		return false, fmt.Errorf("gagal insert outbox event untuk subscription %s: %w", subscriptionID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("gagal commit transaksi expire subscription %s: %w", subscriptionID, err)
	}
	return true, nil
}
