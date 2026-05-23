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
	ID                 uuid.UUID
	UserID             uuid.UUID
	SyncID             string
	Name               string
	LimitIDR           int64
	CategoryID         *uuid.UUID
	PeriodType         BudgetPeriodType
	StartDate          time.Time
	EndDate            *time.Time
	AlertThreshold     int
	DailyLimitEnabled  bool
	IsDeleted          bool
	DeletedAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type BudgetWithProgress struct {
	Budget
	SpentIDR     int64
	CategoryName string
	// Populated only when DailyLimitEnabled = true.
	SpentTodayIDR          int64
	DailyBaseIDR           int64
	CarryoverIDR           int64 // positive = surplus, negative = deficit
	DailyAllowanceTodayIDR int64
	DailyRemainingIDR      int64
}
