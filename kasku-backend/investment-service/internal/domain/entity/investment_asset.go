package entity

import (
	"time"

	"github.com/google/uuid"
)

type AssetType string

const (
	AssetTypeCrypto     AssetType = "CRYPTO"
	AssetTypeGold       AssetType = "GOLD"
	AssetTypeStock      AssetType = "STOCK"
	AssetTypeMutualFund AssetType = "MUTUAL_FUND"
	AssetTypeOther      AssetType = "OTHER"
)

// InvestmentAsset merepresentasikan satu instrumen investasi milik tenant.
type InvestmentAsset struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	AssetType   AssetType  `json:"asset_type"`
	Symbol      string     `json:"symbol"`
	Quantity    float64    `json:"quantity"`
	AvgBuyPrice float64    `json:"avg_buy_price"`
	Currency    string     `json:"currency"`
	IsDeleted   bool       `json:"-"`
	DeletedAt   *time.Time `json:"-"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	// Enriched fields (from price-service, not persisted)
	CurrentPrice *float64 `json:"current_price,omitempty"`
	IsPriceFresh *bool    `json:"is_price_fresh,omitempty"`
}

// UnitHistory merepresentasikan satu entri append-only riwayat perubahan unit.
type UnitHistory struct {
	ID              uuid.UUID `json:"id"`
	AssetID         uuid.UUID `json:"asset_id"`
	TransactionType string    `json:"transaction_type"` // BUY, SELL, ADJUST
	QuantityChange  float64   `json:"quantity_change"`
	PricePerUnit    float64   `json:"price_per_unit"`
	TotalValue      float64   `json:"total_value"` // generated
	Notes           string    `json:"notes"`
	TransactionDate time.Time `json:"transaction_date"`
	RecordedAt      time.Time `json:"recorded_at"`
}
