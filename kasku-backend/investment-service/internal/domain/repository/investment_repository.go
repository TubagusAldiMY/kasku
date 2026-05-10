package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
)

// InvestmentAssetRepository mendefinisikan port untuk akses data instrumen investasi.
// SEMUA method menerima tenantSchema untuk multi-tenancy dan query isolation.
type InvestmentAssetRepository interface {
	CountActive(ctx context.Context, tenantSchema string) (int, error)
	Create(ctx context.Context, tenantSchema string, asset *entity.InvestmentAsset) error
	List(ctx context.Context, tenantSchema string) ([]entity.InvestmentAsset, error)
	GetByID(ctx context.Context, tenantSchema, id string) (*entity.InvestmentAsset, error)
	Update(ctx context.Context, tenantSchema string, asset *entity.InvestmentAsset) error
	SoftDelete(ctx context.Context, tenantSchema, id string) error

	// Unit History
	CreateUnitHistory(ctx context.Context, tenantSchema string, entry *entity.UnitHistory) error
	GetUnitHistory(ctx context.Context, tenantSchema, assetID string, limitMonths int) ([]entity.UnitHistory, error)

	// Update quantity and avg_buy_price after BUY/SELL operation
	UpdateQuantity(ctx context.Context, tenantSchema, assetID string, newQuantity, newAvgBuyPrice float64) error
}
