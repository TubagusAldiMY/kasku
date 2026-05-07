package errors

import "errors"

// DomainError merepresentasikan error bertipe dari domain auth.
// Setiap error memiliki kode mesin (Code) dan pesan ramah pengguna (Message).
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

// Sentinel domain errors — digunakan di use case dan handler untuk branching logic.
var (
	ErrInvalidCredentials = &DomainError{
		Code:    "INVALID_CREDENTIALS",
		Message: "Email atau password salah.",
	}
	ErrAccountLocked = &DomainError{
		Code:    "ACCOUNT_LOCKED",
		Message: "Akun dikunci sementara akibat terlalu banyak percobaan login gagal.",
	}
	ErrAccountNotVerified = &DomainError{
		Code:    "ACCOUNT_NOT_VERIFIED",
		Message: "Akun belum diverifikasi. Silakan cek email Anda.",
	}
	ErrEmailAlreadyExists = &DomainError{
		Code:    "EMAIL_ALREADY_EXISTS",
		Message: "Email sudah terdaftar.",
	}
	ErrUsernameAlreadyExists = &DomainError{
		Code:    "USERNAME_ALREADY_EXISTS",
		Message: "Username sudah digunakan.",
	}
	ErrEmailAlreadyVerified = &DomainError{
		Code:    "EMAIL_ALREADY_VERIFIED",
		Message: "Email sudah terverifikasi.",
	}
	ErrInvalidToken = &DomainError{
		Code:    "INVALID_TOKEN",
		Message: "Token tidak valid atau sudah kadaluwarsa.",
	}
	ErrTokenReuseDetected = &DomainError{
		Code:    "TOKEN_REUSE_DETECTED",
		Message: "Aktivitas mencurigakan terdeteksi. Semua sesi telah dicabut.",
	}
	ErrUserNotFound = &DomainError{
		Code:    "USER_NOT_FOUND",
		Message: "Pengguna tidak ditemukan.",
	}
	ErrPasswordTooShort = &DomainError{
		Code:    "PASSWORD_TOO_SHORT",
		Message: "Password minimal 8 karakter.",
	}
	ErrPasswordTooWeak = &DomainError{
		Code:    "PASSWORD_TOO_WEAK",
		Message: "Password harus mengandung huruf besar, huruf kecil, dan angka.",
	}
	ErrValidation = &DomainError{
		Code:    "VALIDATION_ERROR",
		Message: "Data input tidak valid.",
	}
	ErrServiceUnavailable = &DomainError{
		Code:    "SERVICE_UNAVAILABLE",
		Message: "Layanan sementara tidak tersedia. Silakan coba lagi.",
	}
	ErrInternal = &DomainError{
		Code:    "INTERNAL_ERROR",
		Message: "Terjadi kesalahan internal. Silakan coba lagi.",
	}
)

// IsDomainError memeriksa apakah suatu error merupakan DomainError.
func IsDomainError(err error) (*DomainError, bool) {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr, true
	}
	return nil, false
}
