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
	ErrTransactionNotFound     = &DomainError{Code: "TRANSACTION_NOT_FOUND", Message: "Transaksi tidak ditemukan."}
	ErrCategoryNotFound        = &DomainError{Code: "CATEGORY_NOT_FOUND", Message: "Kategori tidak ditemukan."}
	ErrCategoryHasTransactions = &DomainError{Code: "CATEGORY_HAS_TRANSACTIONS", Message: "Kategori tidak dapat dihapus karena masih memiliki transaksi aktif."}
	ErrTransactionLimitReached = &DomainError{Code: "TRANSACTION_LIMIT_REACHED", Message: "Batas jumlah transaksi bulanan tercapai. Upgrade subscription untuk melanjutkan."}
	ErrExportNotAllowed        = &DomainError{Code: "EXPORT_NOT_ALLOWED", Message: "Ekspor CSV tidak tersedia di plan Anda."}
	ErrInvalidInput            = &DomainError{Code: "INVALID_INPUT", Message: "Input tidak valid."}
	ErrInternal                = &DomainError{Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal."}
)

func IsDomainError(err error) (*DomainError, bool) {
	de, ok := err.(*DomainError)
	return de, ok
}
