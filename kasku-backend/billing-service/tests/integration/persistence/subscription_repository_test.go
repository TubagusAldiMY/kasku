package persistence_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/billing-service/tests/integration"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionRepository_GetByUserID(t *testing.T) {
	pool, _ := integration.SetupPostgres(t)
	repo := persistence.NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()

	t.Run("not found returns ErrSubscriptionNotFound", func(t *testing.T) {
		_, err := repo.GetByUserID(ctx, uuid.NewString())
		assert.ErrorIs(t, err, domainerrors.ErrSubscriptionNotFound)
	})

	t.Run("found returns subscription", func(t *testing.T) {
		userID := uuid.New()
		planID := seedPlan(t, pool, "BASIC")
		seedSubscription(t, pool, userID, planID, entity.StatusActive, time.Now().Add(7*24*time.Hour))

		got, err := repo.GetByUserID(ctx, userID.String())
		require.NoError(t, err)
		assert.Equal(t, userID, got.UserID)
		assert.Equal(t, planID, got.PlanID)
		assert.Equal(t, entity.StatusActive, got.Status)
	})
}

func TestSubscriptionRepository_GetPlanWithLimits(t *testing.T) {
	pool, _ := integration.SetupPostgres(t)
	repo := persistence.NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()

	t.Run("active plan returned with limits", func(t *testing.T) {
		planID := seedPlan(t, pool, "BASIC")
		plan, err := repo.GetPlanWithLimits(ctx, planID.String())
		require.NoError(t, err)
		assert.Equal(t, "BASIC", plan.Name)
		assert.True(t, plan.IsActive)
		assert.Equal(t, int32(500), plan.Limits.MaxTransactionsPerMonth)
	})

	t.Run("inactive plan returns ErrPlanNotFound", func(t *testing.T) {
		planID := seedPlan(t, pool, "BASIC")
		_, err := pool.Exec(ctx, `UPDATE public.subscription_plans SET is_active = false WHERE id = $1`, planID)
		require.NoError(t, err)
		_, err = repo.GetPlanWithLimits(ctx, planID.String())
		assert.ErrorIs(t, err, domainerrors.ErrPlanNotFound)
	})

	t.Run("missing plan returns ErrPlanNotFound", func(t *testing.T) {
		_, err := repo.GetPlanWithLimits(ctx, uuid.NewString())
		assert.ErrorIs(t, err, domainerrors.ErrPlanNotFound)
	})
}

func TestSubscriptionRepository_ListAllPlans(t *testing.T) {
	pool, _ := integration.SetupPostgres(t)
	repo := persistence.NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()

	seedPlan(t, pool, "PRO")   // price 99000
	seedPlan(t, pool, "FREE")  // price 0
	seedPlan(t, pool, "BASIC") // price 49000

	plans, err := repo.ListAllPlans(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(plans), 3)
	// Sorted ASC by price.
	for i := 1; i < len(plans); i++ {
		assert.LessOrEqual(t, plans[i-1].PriceIDR, plans[i].PriceIDR, "plans must be ordered by price ASC")
	}
}

func TestSubscriptionRepository_ListExpiredSubscriptions(t *testing.T) {
	pool, _ := integration.SetupPostgres(t)
	repo := persistence.NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()

	planID := seedPlan(t, pool, "BASIC")
	expiredUser := uuid.New()
	notExpiredUser := uuid.New()
	seedSubscription(t, pool, expiredUser, planID, entity.StatusActive, time.Now().Add(-1*time.Hour))
	seedSubscription(t, pool, notExpiredUser, planID, entity.StatusActive, time.Now().Add(24*time.Hour))

	expired, err := repo.ListExpiredSubscriptions(ctx)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	assert.Equal(t, expiredUser, expired[0].UserID)
}

func TestSubscriptionRepository_ExpireSubscriptionAtomic(t *testing.T) {
	pool, _ := integration.SetupPostgres(t)
	repo := persistence.NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()

	t.Run("flip status + insert outbox in tx", func(t *testing.T) {
		userID := uuid.New()
		planID := seedPlan(t, pool, "BASIC")
		seedSubscription(t, pool, userID, planID, entity.StatusActive, time.Now().Add(-1*time.Hour))

		subID := getSubscriptionID(t, pool, userID)
		payload := []byte(`{"subscription_id":"` + subID.String() + `","user_id":"` + userID.String() + `","plan_name":"BASIC","previous_status":"ACTIVE"}`)

		flipped, err := repo.ExpireSubscriptionAtomic(ctx, subID.String(), "subscription.expired", "subscription.expired", payload)
		require.NoError(t, err)
		assert.True(t, flipped)

		// Status di DB harus EXPIRED.
		var status string
		require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM public.subscriptions WHERE id = $1`, subID).Scan(&status))
		assert.Equal(t, "EXPIRED", status)

		// Outbox event tersimpan dengan routing_key & payload yang benar.
		var routingKey, eventType string
		var payloadOut []byte
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT event_type, routing_key, payload FROM public.outbox_events WHERE payload::text LIKE '%' || $1 || '%' ORDER BY created_at DESC LIMIT 1`,
			subID.String(),
		).Scan(&eventType, &routingKey, &payloadOut))
		assert.Equal(t, "subscription.expired", eventType)
		assert.Equal(t, "subscription.expired", routingKey)
		assert.Contains(t, string(payloadOut), subID.String())
	})

	t.Run("idempotent — re-run returns false", func(t *testing.T) {
		userID := uuid.New()
		planID := seedPlan(t, pool, "BASIC")
		seedSubscription(t, pool, userID, planID, entity.StatusExpired, time.Now().Add(-1*time.Hour))
		subID := getSubscriptionID(t, pool, userID)

		flipped, err := repo.ExpireSubscriptionAtomic(ctx, subID.String(), "subscription.expired", "subscription.expired", []byte(`{}`))
		require.NoError(t, err)
		assert.False(t, flipped)
	})

	t.Run("rollback when subscription not found does not insert outbox", func(t *testing.T) {
		fakeID := uuid.NewString()
		// Count outbox sebelumnya.
		var before int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM public.outbox_events`).Scan(&before))

		flipped, err := repo.ExpireSubscriptionAtomic(ctx, fakeID, "subscription.expired", "subscription.expired", []byte(`{}`))
		require.NoError(t, err)
		assert.False(t, flipped)

		var after int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM public.outbox_events`).Scan(&after))
		assert.Equal(t, before, after, "outbox should not grow on no-op flip")
	})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// seedPlan reuse plan yang sudah di-seed oleh migration 000002 (FREE/BASIC/PRO).
// Pastikan plan aktif (re-activate kalau test sebelumnya men-deactivate).
func seedPlan(t *testing.T, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `UPDATE public.subscription_plans SET is_active = true WHERE name = $1`, name)
	require.NoError(t, err)

	var planID uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT id FROM public.subscription_plans WHERE name = $1`, name).Scan(&planID)
	require.NoError(t, err, "plan %s belum di-seed oleh migration", name)
	return planID
}

func seedSubscription(t *testing.T, pool *pgxpool.Pool, userID, planID uuid.UUID, status entity.SubscriptionStatus, periodEnd time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO public.subscriptions (user_id, plan_id, status, current_period_start, current_period_end)
		 VALUES ($1, $2, $3, now(), $4)`,
		userID, planID, string(status), periodEnd,
	)
	require.NoError(t, err)
}

func getSubscriptionID(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`SELECT id FROM public.subscriptions WHERE user_id = $1`, userID).Scan(&id)
	require.NoError(t, err, "subscription not found for user")
	if errors.Is(err, nil) {
		return id
	}
	return id
}
