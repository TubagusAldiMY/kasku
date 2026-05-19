package event

// UserRegisteredEvent adalah payload event dari auth-service.
type UserRegisteredEvent struct {
	UserID            string `json:"user_id"`
	Email             string `json:"email"`
	Username          string `json:"username"`
	VerificationToken string `json:"verification_token"`
}

// EmailVerificationResentEvent adalah payload event resend verifikasi.
type EmailVerificationResentEvent struct {
	UserID            string `json:"user_id"`
	Email             string `json:"email"`
	VerificationToken string `json:"verification_token"`
}

// PasswordResetRequestedEvent adalah payload event reset password.
type PasswordResetRequestedEvent struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	ResetToken string `json:"reset_token"`
}

// PaymentSucceededEvent adalah payload event pembayaran berhasil.
type PaymentSucceededEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	OrderID   string `json:"order_id"`
	AmountIDR int64  `json:"amount_idr"`
	PlanName  string `json:"plan_name"`
}

// PaymentFailedEvent adalah payload event pembayaran gagal.
type PaymentFailedEvent struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// SubscriptionExpiringEvent adalah payload event subscription akan expire.
type SubscriptionExpiringEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	PlanName  string `json:"plan_name"`
	ExpiresAt string `json:"expires_at"`
}

// SubscriptionExpiredEvent adalah payload event subscription sudah expire.
type SubscriptionExpiredEvent struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	PlanName string `json:"plan_name"`
}

// SubscriptionCancelledEvent adalah payload event subscription dibatalkan oleh user (NOTIF-FR-006).
type SubscriptionCancelledEvent struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	PlanName    string `json:"plan_name"`
	CancelledAt string `json:"cancelled_at"` // RFC3339
}
