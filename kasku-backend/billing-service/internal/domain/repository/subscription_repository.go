package repository

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/google/uuid"
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

	// CreateSubscription membuat subscription record baru untuk user.
	// Jika user sudah memiliki subscription (aktif maupun tidak), implementasi menangani upsert
	// sesuai kebutuhan bisnis: user hanya boleh punya satu subscription per waktu.
	// Status awal selalu ACTIVE dengan current_period_start = now().
	// Mengembalikan ErrActiveSubscriptionExists jika user sudah punya subscription ACTIVE.
	CreateSubscription(ctx context.Context, userID, planID uuid.UUID) (*entity.Subscription, error)

	// ActivateSubscription mengaktifkan subscription setelah pembayaran berhasil dikonfirmasi.
	// Set status=ACTIVE, current_period_end = periodEnd.
	// Dipanggil oleh HandlePaymentWebhookUseCase setelah event payment.success diterima.
	ActivateSubscription(ctx context.Context, subscriptionID uuid.UUID, periodEnd time.Time) error

	// UpgradeSubscription mengupdate plan_id dan memperpanjang period subscription yang sudah ACTIVE.
	// Digunakan saat user upgrade dari FREE ke plan berbayar setelah pembayaran sukses.
	// Set plan_id = newPlanID, current_period_end = periodEnd, updated_at = now().
	// Mengembalikan ErrSubscriptionNotFound jika tidak ada subscription ACTIVE dengan id tersebut.
	UpgradeSubscription(ctx context.Context, subscriptionID, newPlanID uuid.UUID, periodEnd time.Time) error

	// InsertOutboxEvent menyimpan satu event ke tabel outbox_events untuk reliable delivery.
	// Outbox dispatcher akan membaca dan mempublish event ini ke RabbitMQ secara async.
	InsertOutboxEvent(ctx context.Context, eventType, routingKey string, payload []byte) error
}
