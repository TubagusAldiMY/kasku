package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationPreference struct {
	EmailEnabled         bool `json:"email_enabled"`
	PaymentAlertsEnabled bool `json:"payment_alerts_enabled"`
	ExpiryAlertsEnabled  bool `json:"expiry_alerts_enabled"`
}

type PreferenceRepository interface {
	Get(ctx context.Context, userID string) (*NotificationPreference, error)
	Upsert(ctx context.Context, userID string, pref NotificationPreference) (*NotificationPreference, error)
}

type postgresPreferenceRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPreferenceRepository(pool *pgxpool.Pool) PreferenceRepository {
	return &postgresPreferenceRepository{pool: pool}
}

func (r *postgresPreferenceRepository) Get(ctx context.Context, userID string) (*NotificationPreference, error) {
	const query = `
		SELECT email_enabled, payment_alerts_enabled, expiry_alerts_enabled
		FROM public.notification_preferences
		WHERE user_id = $1
	`

	var pref NotificationPreference
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&pref.EmailEnabled,
		&pref.PaymentAlertsEnabled,
		&pref.ExpiryAlertsEnabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mengambil notification preference: %w", err)
	}
	return &pref, nil
}

func (r *postgresPreferenceRepository) Upsert(ctx context.Context, userID string, pref NotificationPreference) (*NotificationPreference, error) {
	const query = `
		INSERT INTO public.notification_preferences
			(user_id, email_enabled, payment_alerts_enabled, expiry_alerts_enabled)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			payment_alerts_enabled = EXCLUDED.payment_alerts_enabled,
			expiry_alerts_enabled = EXCLUDED.expiry_alerts_enabled
		RETURNING email_enabled, payment_alerts_enabled, expiry_alerts_enabled
	`

	var saved NotificationPreference
	if err := r.pool.QueryRow(
		ctx,
		query,
		userID,
		pref.EmailEnabled,
		pref.PaymentAlertsEnabled,
		pref.ExpiryAlertsEnabled,
	).Scan(&saved.EmailEnabled, &saved.PaymentAlertsEnabled, &saved.ExpiryAlertsEnabled); err != nil {
		return nil, fmt.Errorf("gagal menyimpan notification preference: %w", err)
	}
	return &saved, nil
}
