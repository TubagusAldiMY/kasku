package entity

import (
	"time"

	"github.com/google/uuid"
)

type AccountType string

const (
	AccountTypeBank       AccountType = "BANK"
	AccountTypeCash       AccountType = "CASH"
	AccountTypeEwallet    AccountType = "EWALLET"
	AccountTypeInvestment AccountType = "INVESTMENT"
	AccountTypeCreditCard AccountType = "CREDIT_CARD"
	AccountTypeOther      AccountType = "OTHER"
)

type FinancialAccount struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	AccountType AccountType
	Balance     int64 // dalam satuan rupiah (integer, bukan float untuk menghindari floating-point error)
	Currency    string
	Color       string
	Icon        string
	IsDefault   bool
	IsDeleted   bool
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BalanceHistory struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	Amount    int64
	Balance   int64 // saldo setelah perubahan
	Note      string
	CreatedAt time.Time
}
