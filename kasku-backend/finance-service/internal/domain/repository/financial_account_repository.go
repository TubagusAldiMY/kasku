package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
)

// FinancialAccountRepository mendefinisikan port untuk akses data akun keuangan.
// SEMUA method menerima tenantSchema untuk multi-tenancy dan query isolation.
type FinancialAccountRepository interface {
	CountByUserID(ctx context.Context, tenantSchema, userID string) (int, error)
	Create(ctx context.Context, tenantSchema string, account *entity.FinancialAccount) error
	List(ctx context.Context, tenantSchema, userID string) ([]entity.FinancialAccount, error)
	GetByID(ctx context.Context, tenantSchema, id, userID string) (*entity.FinancialAccount, error)
	Update(ctx context.Context, tenantSchema string, account *entity.FinancialAccount) error
	SoftDelete(ctx context.Context, tenantSchema, id, userID string) error
	GetBalanceHistory(ctx context.Context, tenantSchema, accountID string, limitMonths int) ([]entity.BalanceHistory, error)
}
