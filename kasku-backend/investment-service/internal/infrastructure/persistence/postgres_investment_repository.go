package persistence

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/investment-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var tenantSchemaRegex = regexp.MustCompile(`^tenant_[0-9a-f_]{32,36}$`)

// ValidateTenantSchema memvalidasi nama tenant schema untuk mencegah SQL injection.
func ValidateTenantSchema(schema string) error {
	if !tenantSchemaRegex.MatchString(schema) {
		return fmt.Errorf("tenant schema tidak valid: %s", schema)
	}
	return nil
}

type postgresInvestmentRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresInvestmentRepository(pool *pgxpool.Pool) repository.InvestmentAssetRepository {
	return &postgresInvestmentRepository{pool: pool}
}

func (r *postgresInvestmentRepository) CountActive(ctx context.Context, tenantSchema string) (int, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.investment_assets WHERE is_deleted = false", tenantSchema)
	var count int
	if err := r.pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("gagal hitung aset: %w", err)
	}
	return count, nil
}

func (r *postgresInvestmentRepository) Create(ctx context.Context, tenantSchema string, asset *entity.InvestmentAsset) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		INSERT INTO %s.investment_assets
			(id, name, asset_type, symbol, quantity, avg_buy_price, currency, is_deleted, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8, $9, $10)
	`, tenantSchema)
	_, err := r.pool.Exec(ctx, query,
		asset.ID, asset.Name, string(asset.AssetType), asset.Symbol,
		asset.Quantity, asset.AvgBuyPrice, asset.Currency,
		asset.SortOrder, asset.CreatedAt, asset.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert aset: %w", err)
	}
	return nil
}

func (r *postgresInvestmentRepository) List(ctx context.Context, tenantSchema string) ([]entity.InvestmentAsset, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id, name, asset_type, symbol, quantity, avg_buy_price, currency,
		       is_deleted, deleted_at, sort_order, created_at, updated_at
		FROM %s.investment_assets
		WHERE is_deleted = false
		ORDER BY sort_order ASC, created_at ASC
	`, tenantSchema)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query aset: %w", err)
	}
	defer rows.Close()

	var assets []entity.InvestmentAsset
	for rows.Next() {
		a := entity.InvestmentAsset{}
		if err := rows.Scan(
			&a.ID, &a.Name, &a.AssetType, &a.Symbol,
			&a.Quantity, &a.AvgBuyPrice, &a.Currency,
			&a.IsDeleted, &a.DeletedAt, &a.SortOrder,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan aset: %w", err)
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

func (r *postgresInvestmentRepository) GetByID(ctx context.Context, tenantSchema, id string) (*entity.InvestmentAsset, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id, name, asset_type, symbol, quantity, avg_buy_price, currency,
		       is_deleted, deleted_at, sort_order, created_at, updated_at
		FROM %s.investment_assets
		WHERE id = $1 AND is_deleted = false
	`, tenantSchema)

	a := &entity.InvestmentAsset{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.Name, &a.AssetType, &a.Symbol,
		&a.Quantity, &a.AvgBuyPrice, &a.Currency,
		&a.IsDeleted, &a.DeletedAt, &a.SortOrder,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrAssetNotFound
		}
		return nil, fmt.Errorf("gagal get aset: %w", err)
	}
	return a, nil
}

func (r *postgresInvestmentRepository) Update(ctx context.Context, tenantSchema string, asset *entity.InvestmentAsset) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.investment_assets
		SET name = $2, asset_type = $3, symbol = $4, currency = $5, sort_order = $6, updated_at = $7
		WHERE id = $1 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query,
		asset.ID, asset.Name, string(asset.AssetType), asset.Symbol,
		asset.Currency, asset.SortOrder, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("gagal update aset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrAssetNotFound
	}
	return nil
}

func (r *postgresInvestmentRepository) SoftDelete(ctx context.Context, tenantSchema, id string) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.investment_assets
		SET is_deleted = true, deleted_at = now(), updated_at = now()
		WHERE id = $1 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("gagal soft delete aset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrAssetNotFound
	}
	return nil
}

func (r *postgresInvestmentRepository) UpdateQuantity(ctx context.Context, tenantSchema, assetID string, newQuantity, newAvgBuyPrice float64) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.investment_assets
		SET quantity = $2, avg_buy_price = $3, updated_at = now()
		WHERE id = $1 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query, assetID, newQuantity, newAvgBuyPrice)
	if err != nil {
		return fmt.Errorf("gagal update quantity aset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrAssetNotFound
	}
	return nil
}

func (r *postgresInvestmentRepository) CreateUnitHistory(ctx context.Context, tenantSchema string, entry *entity.UnitHistory) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		INSERT INTO %s.unit_history
			(id, asset_id, transaction_type, quantity_change, price_per_unit, notes, transaction_date, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, tenantSchema)
	_, err := r.pool.Exec(ctx, query,
		entry.ID, entry.AssetID, entry.TransactionType,
		entry.QuantityChange, entry.PricePerUnit, entry.Notes,
		entry.TransactionDate, entry.RecordedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert unit history: %w", err)
	}
	return nil
}

func (r *postgresInvestmentRepository) GetUnitHistory(ctx context.Context, tenantSchema, assetID string, limitMonths int) ([]entity.UnitHistory, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}

	var (
		query string
		args  []interface{}
	)

	if limitMonths > 0 {
		query = fmt.Sprintf(`
			SELECT id, asset_id, transaction_type, quantity_change, price_per_unit,
			       ABS(quantity_change) * price_per_unit as total_value,
			       COALESCE(notes, ''), transaction_date, recorded_at
			FROM %s.unit_history
			WHERE asset_id = $1 AND recorded_at >= now() - ($2 || ' months')::interval
			ORDER BY recorded_at DESC
		`, tenantSchema)
		args = []interface{}{assetID, limitMonths}
	} else {
		query = fmt.Sprintf(`
			SELECT id, asset_id, transaction_type, quantity_change, price_per_unit,
			       ABS(quantity_change) * price_per_unit as total_value,
			       COALESCE(notes, ''), transaction_date, recorded_at
			FROM %s.unit_history
			WHERE asset_id = $1
			ORDER BY recorded_at DESC
		`, tenantSchema)
		args = []interface{}{assetID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query unit history: %w", err)
	}
	defer rows.Close()

	var history []entity.UnitHistory
	for rows.Next() {
		h := entity.UnitHistory{}
		if err := rows.Scan(
			&h.ID, &h.AssetID, &h.TransactionType,
			&h.QuantityChange, &h.PricePerUnit, &h.TotalValue,
			&h.Notes, &h.TransactionDate, &h.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan unit history: %w", err)
		}
		history = append(history, h)
	}
	return history, rows.Err()
}
