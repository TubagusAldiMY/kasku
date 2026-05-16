package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresPaymentRepository punya akses ke kasku_billing.payments + subscription_plans.
type postgresPaymentRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPaymentRepository membuat read-only repository untuk kasku_billing.payments.
func NewPostgresPaymentRepository(pool *pgxpool.Pool) repository.PaymentReadRepository {
	return &postgresPaymentRepository{pool: pool}
}

func (r *postgresPaymentRepository) List(ctx context.Context, f repository.PaymentListFilter) ([]entity.PaymentSummary, int64, error) {
	var (
		conds []string
		args  []any
		i     = 1
	)

	if f.UserID != nil {
		conds = append(conds, fmt.Sprintf("p.user_id = $%d", i))
		args = append(args, *f.UserID)
		i++
	}
	if f.Status != nil {
		conds = append(conds, fmt.Sprintf("p.status = $%d", i))
		args = append(args, *f.Status)
		i++
	}
	if f.PlanName != nil {
		conds = append(conds, fmt.Sprintf("plan.name = $%d", i))
		args = append(args, *f.PlanName)
		i++
	}
	if f.From != nil {
		conds = append(conds, fmt.Sprintf("p.created_at >= $%d", i))
		args = append(args, *f.From)
		i++
	}
	if f.To != nil {
		conds = append(conds, fmt.Sprintf("p.created_at < $%d", i))
		args = append(args, *f.To)
		i++
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int64
	countQ := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM public.payments p
		LEFT JOIN public.subscriptions s ON s.id = p.subscription_id
		LEFT JOIN public.subscription_plans plan ON plan.id = s.plan_id
		%s
	`, where)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal count payments: %w", err)
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	listQ := fmt.Sprintf(`
		SELECT p.id, p.user_id, p.order_id, p.amount_idr, p.status,
		       COALESCE(plan.name, '') AS plan_name, p.created_at, p.updated_at
		FROM public.payments p
		LEFT JOIN public.subscriptions s ON s.id = p.subscription_id
		LEFT JOIN public.subscription_plans plan ON plan.id = s.plan_id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, i, i+1)

	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal query payments: %w", err)
	}
	defer rows.Close()

	out := make([]entity.PaymentSummary, 0, limit)
	for rows.Next() {
		var p entity.PaymentSummary
		if err := rows.Scan(&p.ID, &p.UserID, &p.OrderID, &p.AmountIDR, &p.Status, &p.PlanName, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("gagal scan payment: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterasi payments: %w", err)
	}
	return out, total, nil
}

func (r *postgresPaymentRepository) CountMRRActive(ctx context.Context) (int64, error) {
	const q = `
		SELECT COALESCE(SUM(plan.price_idr), 0)::BIGINT
		FROM public.subscriptions s
		JOIN public.subscription_plans plan ON plan.id = s.plan_id
		WHERE s.status = 'ACTIVE'
	`
	var n int64
	if err := r.pool.QueryRow(ctx, q).Scan(&n); err != nil {
		return 0, fmt.Errorf("gagal count MRR: %w", err)
	}
	return n, nil
}

func (r *postgresPaymentRepository) CountByTier(ctx context.Context) (map[string]int64, error) {
	const q = `
		SELECT plan.name, COUNT(s.id)
		FROM public.subscriptions s
		JOIN public.subscription_plans plan ON plan.id = s.plan_id
		WHERE s.status = 'ACTIVE'
		GROUP BY plan.name
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("gagal query tier distribution: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int64)
	for rows.Next() {
		var name string
		var n int64
		if err := rows.Scan(&name, &n); err != nil {
			return nil, fmt.Errorf("gagal scan tier: %w", err)
		}
		out[name] = n
	}
	return out, rows.Err()
}

func (r *postgresPaymentRepository) CountCancelledSince(ctx context.Context, since time.Time) (int64, error) {
	const q = `
		SELECT COUNT(*) FROM public.subscriptions
		WHERE status = 'CANCELLED' AND updated_at >= $1
	`
	var n int64
	if err := r.pool.QueryRow(ctx, q, since).Scan(&n); err != nil {
		return 0, fmt.Errorf("gagal count cancelled subs: %w", err)
	}
	return n, nil
}
