package errors

import "fmt"

// DomainError representasi kesalahan domain yang bisa dipetakan ke HTTP status code.
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// IsDomainError memeriksa apakah error adalah DomainError.
func IsDomainError(err error) (*DomainError, bool) {
	if de, ok := err.(*DomainError); ok {
		return de, true
	}
	return nil, false
}

var (
	ErrAssetNotFound = &DomainError{
		Code:    "ASSET_NOT_FOUND",
		Message: "Instrumen investasi tidak ditemukan.",
	}

	ErrAssetLimitReached = &DomainError{
		Code:    "ASSET_LIMIT_REACHED",
		Message: "Batas jumlah instrumen investasi untuk tier Anda telah tercapai. Upgrade untuk menambah lebih banyak.",
	}

	ErrInvalidInput = &DomainError{
		Code:    "INVALID_INPUT",
		Message: "Input tidak valid.",
	}

	ErrInvalidTransactionType = &DomainError{
		Code:    "INVALID_TRANSACTION_TYPE",
		Message: "Tipe transaksi tidak valid. Gunakan: BUY, SELL, ADJUST.",
	}
)
