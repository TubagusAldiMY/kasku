package errors

// DomainError adalah error bisnis yang membawa kode + pesan ramah klien.
// Pesan tidak boleh membocorkan internal state (alamat service, library trace, dll).
type DomainError struct {
	Code    string
	Message string
}

// Error mengimplementasikan kontrak error.
func (e *DomainError) Error() string {
	return e.Message
}

// Pre-defined domain errors. Pesan singkat, kode konsisten dengan KasKu naming.
var (
	ErrInvalidCredentials = &DomainError{Code: "INVALID_CREDENTIALS", Message: "Username atau password salah."}
	ErrAdminInactive      = &DomainError{Code: "ADMIN_INACTIVE", Message: "Akun admin dinonaktifkan."}
	ErrAdminNotFound      = &DomainError{Code: "ADMIN_NOT_FOUND", Message: "Admin tidak ditemukan."}
	ErrUserNotFound       = &DomainError{Code: "USER_NOT_FOUND", Message: "User tidak ditemukan."}
	ErrSubscriptionNotFound = &DomainError{Code: "SUBSCRIPTION_NOT_FOUND", Message: "Subscription user tidak ditemukan."}
	ErrPlanNotFound       = &DomainError{Code: "PLAN_NOT_FOUND", Message: "Subscription plan tidak ditemukan."}
	ErrInvalidToken       = &DomainError{Code: "INVALID_TOKEN", Message: "Token tidak valid atau kedaluwarsa."}
	ErrTokenRevoked       = &DomainError{Code: "TOKEN_REVOKED", Message: "Token sudah dicabut."}
	ErrUnauthorized       = &DomainError{Code: "UNAUTHORIZED", Message: "Akses ditolak."}
	ErrForbidden          = &DomainError{Code: "FORBIDDEN", Message: "Anda tidak memiliki izin untuk aksi ini."}
	ErrValidation         = &DomainError{Code: "VALIDATION_ERROR", Message: "Format request tidak valid."}
	ErrInternal           = &DomainError{Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal."}
)
