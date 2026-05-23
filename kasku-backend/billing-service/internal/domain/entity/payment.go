package entity

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus merepresentasikan status siklus hidup sebuah payment.
type PaymentStatus string

const (
	// PaymentPending berarti payment sudah diinisiasi ke orchestrator, menunggu konfirmasi user.
	PaymentPending PaymentStatus = "PENDING"

	// PaymentPaid berarti pembayaran berhasil dikonfirmasi via webhook orchestrator.
	PaymentPaid PaymentStatus = "PAID"

	// PaymentFailed berarti pembayaran gagal (ditolak, saldo tidak cukup, dll).
	PaymentFailed PaymentStatus = "FAILED"

	// PaymentExpired berarti window pembayaran habis sebelum user menyelesaikan transaksi.
	PaymentExpired PaymentStatus = "EXPIRED"
)

// IsFinalStatus mengembalikan true jika status adalah terminal (tidak akan berubah lagi).
// Digunakan untuk idempotency check pada webhook handler.
func (s PaymentStatus) IsFinalStatus() bool {
	return s == PaymentPaid || s == PaymentFailed || s == PaymentExpired
}

// PaymentMethod mendefinisikan metode pembayaran yang didukung oleh Payment Orchestrator.
type PaymentMethod string

const (
	// PaymentMethodQRIS menggunakan QR Code Indonesia Standard.
	PaymentMethodQRIS PaymentMethod = "QRIS"

	// PaymentMethodVirtualAccount menggunakan transfer ke nomor virtual account.
	PaymentMethodVirtualAccount PaymentMethod = "VIRTUAL_ACCOUNT"
)

// DefaultPaymentMethod adalah metode default jika tidak dispesifikasikan oleh user.
const DefaultPaymentMethod = PaymentMethodQRIS

// ParsePaymentMethod mengkonversi string ke PaymentMethod dengan validasi.
// Mengembalikan DefaultPaymentMethod jika input kosong.
func ParsePaymentMethod(raw string) (PaymentMethod, bool) {
	switch PaymentMethod(raw) {
	case PaymentMethodQRIS, PaymentMethodVirtualAccount:
		return PaymentMethod(raw), true
	case "":
		return DefaultPaymentMethod, true
	default:
		return "", false
	}
}

// Payment merepresentasikan satu transaksi pembayaran yang diproses melalui Payment Orchestrator.
//
// Lifecycle: PENDING → PAID (sukses) | FAILED | EXPIRED
// SubscriptionID nullable karena subscription baru dibuat setelah payment berhasil (PAID).
// PlanID disimpan di sini agar webhook handler tidak perlu lookup tambahan ke tabel lain.
type Payment struct {
	ID                 uuid.UUID
	SubscriptionID     *uuid.UUID    // nullable — diisi setelah subscription diaktivasi
	UserID             uuid.UUID
	PlanID             uuid.UUID     // disimpan saat create agar tersedia di webhook handler
	OrderID            string        // internal reference = ExternalRefID (KASKU-SUB-{userID}-{ts})
	AmountIDR          int
	DurationDays       int           // 30 untuk bulanan, 365 untuk tahunan
	Status             PaymentStatus
	PaymentMethod      PaymentMethod
	PaymentURL         string        // QRIS URL atau VA info untuk ditampilkan di frontend
	ExternalPaymentID  string        // ID payment dari Payment Orchestrator
	ExternalRefID      string        // refId yang kita kirimkan ke orchestrator (= OrderID)
	ExpiresAt          *time.Time    // batas waktu pembayaran dari orchestrator
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
