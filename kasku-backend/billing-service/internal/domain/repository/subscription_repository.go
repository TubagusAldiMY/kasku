package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
)

// SubscriptionRepository mendefinisikan kontrak akses data untuk subscription dan plan.
// Semua implementasi wajib menggunakan parameterized query untuk mencegah SQL injection.
//
//go:generate mockgen -source=$GOFILE -destination=../../../tests/mocks/mock_subscription_repository.go -package=mocks
type SubscriptionRepository interface {
	// GetByUserID mengembalikan subscription aktif milik user.
	// Mengembalikan ErrSubscriptionNotFound jika tidak ada.
	GetByUserID(ctx context.Context, userID string) (*entity.Subscription, error)

	// GetPlanWithLimits mengembalikan plan beserta tier limitsnya.
	// Mengembalikan ErrPlanNotFound jika tidak ada atau plan tidak aktif.
	GetPlanWithLimits(ctx context.Context, planID string) (*entity.SubscriptionPlan, error)

	// ListAllPlans mengembalikan semua plan yang aktif, diurutkan berdasarkan harga.
	ListAllPlans(ctx context.Context) ([]entity.SubscriptionPlan, error)

	// ListExpiredSubscriptions mengembalikan semua subscription yang sudah melewati current_period_end.
	ListExpiredSubscriptions(ctx context.Context) ([]entity.Subscription, error)

	// UpdateStatus mengubah status subscription berdasarkan subscriptionID.
	UpdateStatus(ctx context.Context, subscriptionID string, status entity.SubscriptionStatus) error

	// ExpireSubscriptionAtomic melakukan dua operasi dalam satu transaksi:
	//   1) UPDATE subscriptions SET status='EXPIRED' WHERE id=? AND status='ACTIVE'
	//   2) INSERT INTO outbox_events (event_type, routing_key, payload)
	// Mengembalikan (true, nil) bila status berhasil di-flip, (false, nil) bila
	// subscription sudah tidak ACTIVE (idempotent saat cron re-run).
	ExpireSubscriptionAtomic(
		ctx context.Context,
		subscriptionID string,
		eventType string,
		routingKey string,
		payload []byte,
	) (bool, error)
}
