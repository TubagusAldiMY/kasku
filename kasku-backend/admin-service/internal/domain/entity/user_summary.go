package entity

import (
	"time"

	"github.com/google/uuid"
)

// UserSummary adalah read-model untuk daftar user di dashboard admin.
// Field di-merge dari kasku_auth.users + kasku_billing.subscriptions (in-memory join).
type UserSummary struct {
	// Dari kasku_auth.users
	ID            uuid.UUID
	Email         string
	Username      string
	IsActive      bool
	EmailVerified bool
	CreatedAt     time.Time
	LastLoginAt   *time.Time

	// Dari kasku_billing — di-merge oleh use case
	SubscriptionTier   string // "FREE" | "BASIC" | "PRO" | "" kalau belum di-merge
	SubscriptionStatus string // "ACTIVE" | "CANCELLED" | ...
}

// UserDetail adalah view rinci untuk admin detail page.
// Termasuk subscription history + (opsional) usage stats dari tenant schema.
type UserDetail struct {
	UserSummary

	SubscriptionID            *uuid.UUID
	SubscriptionStartedAt     *time.Time
	SubscriptionEndsAt        *time.Time
	SubscriptionPriceIDR      int64

	// Usage stats — kosong di MVP karena network isolation;
	// di-populate dari tenant schema kalau admin-service punya akses ke kasku_finance.
	TotalTransactions *int64
	TotalAccounts     *int64
	TotalInvestments  *int64
}
