package repository

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/google/uuid"
)

// PaymentListFilter adalah opsi filter untuk list payments di kasku_billing.
type PaymentListFilter struct {
	UserID   *uuid.UUID
	Status   *string
	PlanName *string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

// PaymentReadRepository adalah port untuk membaca payments dari kasku_billing.
type PaymentReadRepository interface {
	List(ctx context.Context, filter PaymentListFilter) ([]entity.PaymentSummary, int64, error)
	// CountMRRActive menghitung Monthly Recurring Revenue dari subscriptions aktif.
	CountMRRActive(ctx context.Context) (int64, error)
	// CountByTier mengembalikan tier distribution (PRO/BASIC/FREE → count).
	CountByTier(ctx context.Context) (map[string]int64, error)
	// CountCancelledLast30Days untuk churn calculation.
	CountCancelledSince(ctx context.Context, since time.Time) (int64, error)
}
