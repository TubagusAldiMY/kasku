package entity

import (
	"time"

	"github.com/google/uuid"
)

type BudgetPeriodType string

const (
	PeriodMonthly BudgetPeriodType = "MONTHLY"
	PeriodWeekly  BudgetPeriodType = "WEEKLY"
	PeriodCustom  BudgetPeriodType = "CUSTOM"
)

type Budget struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	SyncID         string
	Name           string
	LimitIDR       int64
	CategoryID     *uuid.UUID
	PeriodType     BudgetPeriodType
	StartDate      time.Time
	EndDate        *time.Time
	AlertThreshold int
	IsDeleted      bool
	DeletedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type BudgetWithProgress struct {
	Budget
	SpentIDR     int64
	CategoryName string
}
