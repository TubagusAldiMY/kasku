package dto

import "time"

// UserListItem adalah baris satu user di list response.
type UserListItem struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	Username           string     `json:"username"`
	IsActive           bool       `json:"is_active"`
	EmailVerified      bool       `json:"email_verified"`
	SubscriptionTier   string     `json:"subscription_tier"`
	SubscriptionStatus string     `json:"subscription_status"`
	CreatedAt          time.Time  `json:"created_at"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
}

// UserDetailDTO membawa detail lengkap user untuk halaman detail admin.
type UserDetailDTO struct {
	UserListItem
	SubscriptionID        *string    `json:"subscription_id,omitempty"`
	SubscriptionStartedAt *time.Time `json:"subscription_started_at,omitempty"`
	SubscriptionEndsAt    *time.Time `json:"subscription_ends_at,omitempty"`
	SubscriptionPriceIDR  int64      `json:"subscription_price_idr"`
}

// SuspendRequest body untuk POST /v1/admin/users/:id/suspend.
type SuspendRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=500"`
}

// ActivateRequest body untuk POST /v1/admin/users/:id/activate.
type ActivateRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=500"`
}

// OverrideSubscriptionRequest body untuk POST /v1/admin/users/:id/override-subscription.
type OverrideSubscriptionRequest struct {
	PlanName string `json:"plan_name" binding:"required,oneof=FREE BASIC PRO"`
	Reason   string `json:"reason" binding:"required,min=3,max=500"`
}

// PaginationMeta dipakai semua list response.
type PaginationMeta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}
