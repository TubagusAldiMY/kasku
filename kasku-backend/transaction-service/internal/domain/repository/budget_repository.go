package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
)

type BudgetRepository interface {
	Count(ctx context.Context, tenantSchema, userID string) (int, error)
	Create(ctx context.Context, tenantSchema string, b *entity.Budget) error
	List(ctx context.Context, tenantSchema, userID string) ([]entity.BudgetWithProgress, error)
	GetByID(ctx context.Context, tenantSchema, id, userID string) (*entity.BudgetWithProgress, error)
	Update(ctx context.Context, tenantSchema string, b *entity.Budget) error
	SoftDelete(ctx context.Context, tenantSchema, id, userID string) error
}
