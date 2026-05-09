package errors

import "fmt"

type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

var (
	ErrAccountNotFound     = &DomainError{Code: "ACCOUNT_NOT_FOUND", Message: "Akun keuangan tidak ditemukan."}
	ErrAccountLimitReached = &DomainError{Code: "ACCOUNT_LIMIT_REACHED", Message: "Batas jumlah akun keuangan tercapai. Upgrade subscription untuk menambah lebih banyak akun."}
	ErrInvalidInput        = &DomainError{Code: "INVALID_INPUT", Message: "Input tidak valid."}
	ErrInternal            = &DomainError{Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal."}
)

func IsDomainError(err error) (*DomainError, bool) {
	de, ok := err.(*DomainError)
	return de, ok
}
