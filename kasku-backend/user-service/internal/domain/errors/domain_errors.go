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
	ErrUserNotFound = &DomainError{Code: "USER_NOT_FOUND", Message: "User tidak ditemukan."}
	ErrInternal     = &DomainError{Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal."}
)

func IsDomainError(err error) bool {
	_, ok := err.(*DomainError)
	return ok
}
