package errors

import (
	"errors"
	"fmt"
)

type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

var (
	ErrTransactionNotFound            = &DomainError{Code: "TRANSACTION_NOT_FOUND", Message: "Transaksi tidak ditemukan."}
	ErrCategoryNotFound               = &DomainError{Code: "CATEGORY_NOT_FOUND", Message: "Kategori tidak ditemukan."}
	ErrCategoryHasTransactions        = &DomainError{Code: "CATEGORY_HAS_TRANSACTIONS", Message: "Kategori tidak dapat dihapus karena masih memiliki transaksi aktif."}
	ErrDefaultCategoryCannotBeDeleted = &DomainError{Code: "DEFAULT_CATEGORY_CANNOT_BE_DELETED", Message: "Kategori default tidak dapat dihapus."}
	ErrTransactionLimitReached        = &DomainError{Code: "TRANSACTION_LIMIT_REACHED", Message: "Batas jumlah transaksi bulanan tercapai. Upgrade subscription untuk melanjutkan."}
	ErrExportNotAllowed               = &DomainError{Code: "EXPORT_NOT_ALLOWED", Message: "Ekspor CSV tidak tersedia di plan Anda."}
	ErrInvalidInput                   = &DomainError{Code: "INVALID_INPUT", Message: "Input tidak valid."}
	ErrInternal                       = &DomainError{Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal."}
	ErrInsufficientBalance            = &DomainError{Code: "INSUFFICIENT_BALANCE", Message: "Saldo rekening asal tidak mencukupi untuk melakukan transfer."}
	ErrAccountNotFound                = &DomainError{Code: "ACCOUNT_NOT_FOUND", Message: "Rekening tidak ditemukan."}
	ErrBudgetNotFound                 = &DomainError{Code: "BUDGET_NOT_FOUND", Message: "Anggaran tidak ditemukan."}
	ErrBudgetLimitReached             = &DomainError{Code: "BUDGET_LIMIT_REACHED", Message: "Batas jumlah anggaran tercapai. Upgrade subscription untuk melanjutkan."}
)

func IsDomainError(err error) (*DomainError, bool) {
	var de *DomainError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}
