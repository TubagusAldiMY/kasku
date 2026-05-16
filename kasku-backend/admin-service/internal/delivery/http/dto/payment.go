package dto

import "time"

// PaymentListItem adalah baris satu payment di list response.
type PaymentListItem struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	OrderID    string    `json:"order_id"`
	AmountIDR  int64     `json:"amount_idr"`
	Status     string    `json:"status"`
	PlanName   string    `json:"plan_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
