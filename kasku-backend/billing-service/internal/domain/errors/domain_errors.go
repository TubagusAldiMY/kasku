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
	ErrSubscriptionNotFound = &DomainError{Code: "SUBSCRIPTION_NOT_FOUND", Message: "Subscription tidak ditemukan."}
	ErrPlanNotFound         = &DomainError{Code: "PLAN_NOT_FOUND", Message: "Subscription plan tidak ditemukan."}
	ErrInternal             = &DomainError{Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal."}
)

// IsDomainError mengembalikan true jika err adalah instance *DomainError.
func IsDomainError(err error) bool {
	_, ok := err.(*DomainError)
	return ok
}
