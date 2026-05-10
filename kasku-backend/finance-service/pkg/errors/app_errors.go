package errors

import "fmt"

// ErrorType mendefinisikan klasifikasi error untuk mapping ke HTTP?gRPC status.
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "VALIDATION_ERROR"
	ErrorTypeInternal   ErrorType = "INTERNAL_ERROR"
	ErrorTypeConflict   ErrorType = "CONFLICT_ERROR"
)

// AppError adalah custom error yang memenuhu interface error bawaan Go.
type AppError struct {
	Type    ErrorType
	Message string
	Err     error // Original Error  Untuk Tracking
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Helper untuk Inisialisasi error
func NewInternalError(msg string, err error) *AppError {
	return &AppError{Type: ErrorTypeInternal, Message: msg, Err: err}
}
