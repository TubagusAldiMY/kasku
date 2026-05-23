package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/google/uuid"
)

// PaymentRepository mendefinisikan kontrak akses data untuk entitas Payment.
// Semua implementasi wajib menggunakan parameterized query untuk mencegah SQL injection.
//
//go:generate mockgen -source=$GOFILE -destination=../../../tests/mocks/mock_payment_repository.go -package=mocks
type PaymentRepository interface {
	// Create menyimpan payment baru ke database.
	// OrderID harus unik — akan error jika sudah ada payment dengan OrderID yang sama.
	Create(ctx context.Context, p *entity.Payment) error

	// GetByOrderID mengambil payment berdasarkan internal order ID (KASKU-SUB-...).
	// Mengembalikan ErrPaymentNotFound jika tidak ada.
	GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error)

	// GetByExternalRefID mengambil payment berdasarkan refId yang dikirim ke orchestrator.
	// Digunakan oleh webhook handler untuk lookup idempotency.
	// Mengembalikan ErrPaymentNotFound jika tidak ada.
	GetByExternalRefID(ctx context.Context, externalRefID string) (*entity.Payment, error)

	// UpdateStatus mengubah status payment dan mencatat external_payment_id dari orchestrator.
	// externalPaymentID boleh kosong (misal saat transisi ke EXPIRED).
	UpdateStatus(ctx context.Context, paymentID uuid.UUID, status entity.PaymentStatus, externalPaymentID string) error

	// LinkToSubscription mengisi kolom subscription_id setelah subscription berhasil diaktivasi.
	// Dipanggil setelah CreateSubscription + ActivateSubscription selesai.
	LinkToSubscription(ctx context.Context, paymentID, subscriptionID uuid.UUID) error
}
