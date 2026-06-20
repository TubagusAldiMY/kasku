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

type CategoryBreakdown struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	TotalAmount  int64   `json:"total_amount"`
	Count        int     `json:"count"`
	Percentage   float64 `json:"percentage"`
}

type MonthlyPoint struct {
	Month   string `json:"month"`
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
}

type ReportData struct {
	PeriodFrom        string              `json:"period_from"`
	PeriodTo          string              `json:"period_to"`
	TotalIncome       int64               `json:"total_income"`
	TotalExpense      int64               `json:"total_expense"`
	NetAmount         int64               `json:"net_amount"`
	CategoryBreakdown []CategoryBreakdown `json:"category_breakdown"`
	MonthlyTrend      []MonthlyPoint      `json:"monthly_trend"`
}
