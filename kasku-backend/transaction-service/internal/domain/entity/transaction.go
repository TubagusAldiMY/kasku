package entity

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionIncome   TransactionType = "INCOME"
	TransactionExpense  TransactionType = "EXPENSE"
	TransactionTransfer TransactionType = "TRANSFER"
)

type Transaction struct {
	ID              uuid.UUID
	SyncID          string
	AccountID       uuid.UUID
	CategoryID      *uuid.UUID
	BudgetID        *uuid.UUID
	TransactionType TransactionType
	AmountIDR       int64
	TransactionDate time.Time
	Notes           string
	ToAccountID     *uuid.UUID
	IsDeleted       bool
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TransactionSummary struct {
	TotalIncome  int64
	TotalExpense int64
	NetAmount    int64
}
