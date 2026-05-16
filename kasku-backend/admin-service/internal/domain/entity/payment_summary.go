package entity

import (
	"time"

	"github.com/google/uuid"
)

// PaymentSummary adalah read-model untuk daftar pembayaran di admin dashboard.
// Field berasal dari kasku_billing.payments.
type PaymentSummary struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	OrderID    string
	AmountIDR  int64
	Status     string // PENDING | SUCCESS | FAILED | EXPIRED | REFUNDED
	PlanName   string // FREE | BASIC | PRO — joined dari subscription_plans
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// DashboardStats adalah aggregate metric untuk halaman dashboard admin (F-ADM-02).
type DashboardStats struct {
	TotalUsers         int64
	TotalActiveUsers   int64
	TierDistribution   map[string]int64 // {"FREE": 120, "BASIC": 30, "PRO": 5}
	MRRIDR             int64            // Monthly Recurring Revenue (Rupiah)
	ChurnRate30dPct    float64          // 0..100
	NewUsersLast7Days  int64
}
