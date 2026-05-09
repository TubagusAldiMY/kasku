package entity

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus merepresentasikan status aktif sebuah subscription.
type SubscriptionStatus string

const (
	StatusActive    SubscriptionStatus = "ACTIVE"
	StatusExpired   SubscriptionStatus = "EXPIRED"
	StatusCancelled SubscriptionStatus = "CANCELLED"
)

// SubscriptionPlan adalah paket berlangganan yang tersedia (FREE, BASIC, PRO).
type SubscriptionPlan struct {
	ID       uuid.UUID
	Name     string
	PriceIDR int
	Limits   PlanLimits
	IsActive bool
}

// PlanLimits mendefinisikan batas-batas penggunaan fitur untuk satu tier subscription.
// Nilai -1 berarti unlimited (khusus untuk tier PRO).
type PlanLimits struct {
	MaxTransactionsPerMonth   int32 `json:"MaxTransactionsPerMonth"`
	MaxFinancialAccounts      int32 `json:"MaxFinancialAccounts"`
	MaxInvestmentInstruments  int32 `json:"MaxInvestmentInstruments"`
	HistoryRetentionMonths    int32 `json:"HistoryRetentionMonths"`
	EmailNotificationsEnabled bool  `json:"EmailNotificationsEnabled"`
	ExportCsvEnabled          bool  `json:"ExportCsvEnabled"`
}

// Subscription merepresentasikan langganan aktif milik seorang user.
type Subscription struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	PlanID             uuid.UUID
	Status             SubscriptionStatus
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
