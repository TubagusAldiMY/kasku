package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresPaymentRepository mengimplementasikan repository.PaymentRepository
// menggunakan pgx/v5 connection pool ke database kasku_billing.
type postgresPaymentRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPaymentRepository membuat instance payment repository baru.
func NewPostgresPaymentRepository(pool *pgxpool.Pool) repository.PaymentRepository {
	return &postgresPaymentRepository{pool: pool}
}

// Create menyimpan payment baru ke database.
// Semua kolom wajib diisi sebelum memanggil Create — subscription_id boleh nil.
func (r *postgresPaymentRepository) Create(ctx context.Context, p *entity.Payment) error {
	const query = `
		INSERT INTO public.payments (
			id, subscription_id, user_id, plan_id, order_id, amount_idr, duration_days,
			status, payment_method, payment_url,
			external_payment_id, external_ref_id, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12, $13
		)
	`
	_, err := r.pool.Exec(ctx, query,
		p.ID,
		p.SubscriptionID,
		p.UserID,
		p.PlanID,
		p.OrderID,
		p.AmountIDR,
		p.DurationDays,
		string(p.Status),
		string(p.PaymentMethod),
		p.PaymentURL,
		p.ExternalPaymentID,
		p.ExternalRefID,
		p.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert payment %s: %w", p.ID, err)
	}
	return nil
}

// GetByOrderID mengambil payment berdasarkan internal order ID.
func (r *postgresPaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error) {
	const query = `
		SELECT id, subscription_id, user_id, plan_id, order_id, amount_idr, duration_days,
		       status, payment_method, payment_url,
		       external_payment_id, external_ref_id, expires_at,
		       created_at, updated_at
		FROM public.payments
		WHERE order_id = $1
		LIMIT 1
	`
	return r.scanPayment(r.pool.QueryRow(ctx, query, orderID))
}

// GetByExternalRefID mengambil payment berdasarkan refId yang dikirim ke orchestrator.
// Digunakan oleh webhook handler untuk lookup dan idempotency check.
func (r *postgresPaymentRepository) GetByExternalRefID(ctx context.Context, externalRefID string) (*entity.Payment, error) {
	const query = `
		SELECT id, subscription_id, user_id, plan_id, order_id, amount_idr, duration_days,
		       status, payment_method, payment_url,
		       external_payment_id, external_ref_id, expires_at,
		       created_at, updated_at
		FROM public.payments
		WHERE external_ref_id = $1
		LIMIT 1
	`
	return r.scanPayment(r.pool.QueryRow(ctx, query, externalRefID))
}

// UpdateStatus mengubah status payment dan mencatat external_payment_id dari orchestrator.
// externalPaymentID boleh string kosong saat transisi ke EXPIRED/FAILED tanpa ID dari orchestrator.
func (r *postgresPaymentRepository) UpdateStatus(
	ctx context.Context,
	paymentID uuid.UUID,
	status entity.PaymentStatus,
	externalPaymentID string,
) error {
	const query = `
		UPDATE public.payments
		SET status = $2,
		    external_payment_id = CASE WHEN $3 != '' THEN $3 ELSE external_payment_id END,
		    updated_at = now()
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, paymentID, string(status), externalPaymentID)
	if err != nil {
		return fmt.Errorf("gagal update status payment %s: %w", paymentID, err)
	}
	if tag.RowsAffected() == 0 {
		return domainerrors.ErrPaymentNotFound
	}
	return nil
}

// LinkToSubscription mengisi kolom subscription_id setelah subscription berhasil diaktivasi.
func (r *postgresPaymentRepository) LinkToSubscription(
	ctx context.Context,
	paymentID, subscriptionID uuid.UUID,
) error {
	const query = `
		UPDATE public.payments
		SET subscription_id = $2, updated_at = now()
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, paymentID, subscriptionID)
	if err != nil {
		return fmt.Errorf("gagal link payment %s ke subscription %s: %w", paymentID, subscriptionID, err)
	}
	if tag.RowsAffected() == 0 {
		return domainerrors.ErrPaymentNotFound
	}
	return nil
}

// scanPayment mem-scan satu baris query result ke entitas Payment.
// Menangani nullable subscription_id dengan pointer.
func (r *postgresPaymentRepository) scanPayment(row pgx.Row) (*entity.Payment, error) {
	p := &entity.Payment{}
	var statusStr string
	var methodStr string
	var externalPaymentID *string // nullable — hanya NULL jika payment orchestrator belum merespons

	err := row.Scan(
		&p.ID,
		&p.SubscriptionID,
		&p.UserID,
		&p.PlanID,
		&p.OrderID,
		&p.AmountIDR,
		&p.DurationDays,
		&statusStr,
		&methodStr,
		&p.PaymentURL,
		&externalPaymentID,
		&p.ExternalRefID,
		&p.ExpiresAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if externalPaymentID != nil {
		p.ExternalPaymentID = *externalPaymentID
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("gagal scan baris payment: %w", err)
	}

	p.Status = entity.PaymentStatus(statusStr)
	p.PaymentMethod = entity.PaymentMethod(methodStr)
	return p, nil
}
