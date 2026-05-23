package repository

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
)

// TransactionRepository mendefinisikan port akses data transaksi (multi-tenant).
type TransactionRepository interface {
	CountMonthly(ctx context.Context, tenantSchema, userID string, month time.Time) (int, error)
	Create(ctx context.Context, tenantSchema string, tx *entity.Transaction) error
	List(ctx context.Context, tenantSchema, userID string, from, to time.Time) ([]entity.Transaction, error)
	GetByID(ctx context.Context, tenantSchema, id, userID string) (*entity.Transaction, error)
	Update(ctx context.Context, tenantSchema, userID string, tx *entity.Transaction) error
	SoftDelete(ctx context.Context, tenantSchema, id, userID string) error
	GetSummary(ctx context.Context, tenantSchema, userID string, from, to time.Time) (*entity.TransactionSummary, error)
	ListForExport(ctx context.Context, tenantSchema, userID string, from, to *time.Time) ([]entity.Transaction, error)
}

// CategoryRepository mendefinisikan port akses data kategori (multi-tenant).
type CategoryRepository interface {
	List(ctx context.Context, tenantSchema string) ([]entity.Category, error)
	GetByID(ctx context.Context, tenantSchema, id string) (*entity.Category, error)
	Create(ctx context.Context, tenantSchema string, cat *entity.Category) error
	Update(ctx context.Context, tenantSchema string, cat *entity.Category) error
	SoftDelete(ctx context.Context, tenantSchema, id string) error
	HasActiveTransactions(ctx context.Context, tenantSchema, categoryID string) (bool, error)
}
