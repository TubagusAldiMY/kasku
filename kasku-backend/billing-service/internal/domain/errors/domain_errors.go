package errors

import "fmt"

// DomainError adalah typed error untuk kesalahan yang berasal dari domain business logic.
// Setiap error memiliki kode unik untuk memudahkan mapping ke HTTP response di delivery layer.
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Sentinel domain errors — gunakan perbandingan langsung (err == ErrXxx) atau IsDomainError.
var (
	ErrSubscriptionNotFound    = &DomainError{Code: "SUBSCRIPTION_NOT_FOUND", Message: "Subscription tidak ditemukan."}
	ErrPlanNotFound            = &DomainError{Code: "PLAN_NOT_FOUND", Message: "Subscription plan tidak ditemukan."}
	ErrInternal                = &DomainError{Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal."}
	ErrPaymentNotFound         = &DomainError{Code: "PAYMENT_NOT_FOUND", Message: "Data pembayaran tidak ditemukan."}
	ErrPaymentAlreadyProcessed = &DomainError{Code: "PAYMENT_ALREADY_PROCESSED", Message: "Pembayaran sudah diproses sebelumnya."}
	ErrActiveSubscriptionExists = &DomainError{Code: "ACTIVE_SUBSCRIPTION_EXISTS", Message: "User sudah memiliki subscription aktif."}
	ErrPaymentGatewayUnavailable = &DomainError{Code: "PAYMENT_GATEWAY_UNAVAILABLE", Message: "Layanan pembayaran sedang tidak tersedia, silakan coba beberapa saat lagi."}
	ErrInvalidPaymentMethod    = &DomainError{Code: "INVALID_PAYMENT_METHOD", Message: "Metode pembayaran tidak valid. Gunakan QRIS atau VIRTUAL_ACCOUNT."}
	ErrInvalidWebhookSignature = &DomainError{Code: "INVALID_WEBHOOK_SIGNATURE", Message: "Tanda tangan webhook tidak valid."}
)

// IsDomainError mengembalikan true jika err adalah instance *DomainError.
func IsDomainError(err error) bool {
	_, ok := err.(*DomainError)
	return ok
}
