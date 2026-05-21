package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresBudgetRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBudgetRepository(pool *pgxpool.Pool) repository.BudgetRepository {
	return &postgresBudgetRepository{pool: pool}
}

func (r *postgresBudgetRepository) Count(ctx context.Context, tenantSchema, userID string) (int, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return 0, err
	}
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s.budgets WHERE user_id = $1 AND is_deleted = false",
		tenantSchema,
	)
	var count int
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("gagal hitung anggaran: %w", err)
	}
	return count, nil
}

func (r *postgresBudgetRepository) Create(ctx context.Context, tenantSchema string, b *entity.Budget) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		INSERT INTO %s.budgets
			(id, user_id, sync_id, name, limit_idr, category_id, period_type,
			 start_date, end_date, alert_threshold, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false, $11, $12)
	`, tenantSchema)
	_, err := r.pool.Exec(ctx, query,
		b.ID, b.UserID, b.SyncID, b.Name, b.LimitIDR, b.CategoryID, string(b.PeriodType),
		b.StartDate, b.EndDate, b.AlertThreshold, b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert anggaran: %w", err)
	}
	return nil
}

const budgetSelectCols = `
	b.id, b.user_id, b.sync_id, b.name, b.limit_idr, b.category_id,
	b.period_type, b.start_date, b.end_date, b.alert_threshold,
	b.is_deleted, b.deleted_at, b.created_at, b.updated_at,
	COALESCE(c.name, '') AS category_name,
	COALESCE((
		SELECT SUM(t.amount_idr)
		FROM %[1]s.transactions t
		WHERE t.is_deleted = false
		  AND t.transaction_type = 'EXPENSE'
		  AND (b.category_id IS NULL OR t.category_id = b.category_id)
		  AND t.account_id IN (
		      SELECT id FROM %[1]s.financial_accounts
		      WHERE user_id = $1 AND is_deleted = false
		  )
		  AND t.transaction_date >= CASE b.period_type
		        WHEN 'MONTHLY' THEN date_trunc('month', CURRENT_DATE)::date
		        WHEN 'WEEKLY'  THEN date_trunc('week', CURRENT_DATE)::date
		        ELSE b.start_date
		      END
		  AND t.transaction_date < CASE b.period_type
		        WHEN 'MONTHLY' THEN (date_trunc('month', CURRENT_DATE) + interval '1 month')::date
		        WHEN 'WEEKLY'  THEN (date_trunc('week', CURRENT_DATE) + interval '1 week')::date
		        ELSE COALESCE(b.end_date + 1, CURRENT_DATE + 1)
		      END
	), 0) AS spent_idr
`

func (r *postgresBudgetRepository) List(ctx context.Context, tenantSchema, userID string) ([]entity.BudgetWithProgress, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	cols := fmt.Sprintf(budgetSelectCols, tenantSchema)
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.budgets b
		LEFT JOIN %s.categories c ON c.id = b.category_id AND c.is_deleted = false
		WHERE b.user_id = $1 AND b.is_deleted = false
		ORDER BY b.created_at ASC
	`, cols, tenantSchema, tenantSchema)

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal query anggaran: %w", err)
	}
	defer rows.Close()

	var budgets []entity.BudgetWithProgress
	for rows.Next() {
		b, err := scanBudgetRow(rows)
		if err != nil {
			return nil, fmt.Errorf("gagal scan anggaran: %w", err)
		}
		budgets = append(budgets, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi baris anggaran: %w", err)
	}
	return budgets, nil
}

func (r *postgresBudgetRepository) GetByID(ctx context.Context, tenantSchema, id, userID string) (*entity.BudgetWithProgress, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	cols := fmt.Sprintf(budgetSelectCols, tenantSchema)
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.budgets b
		LEFT JOIN %s.categories c ON c.id = b.category_id AND c.is_deleted = false
		WHERE b.id = $2 AND b.user_id = $1 AND b.is_deleted = false
	`, cols, tenantSchema, tenantSchema)

	row := r.pool.QueryRow(ctx, query, userID, id)
	b, err := scanBudgetRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrBudgetNotFound
		}
		return nil, fmt.Errorf("gagal get anggaran: %w", err)
	}
	return &b, nil
}

func (r *postgresBudgetRepository) Update(ctx context.Context, tenantSchema string, b *entity.Budget) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.budgets
		SET name = $3, limit_idr = $4, category_id = $5, alert_threshold = $6, updated_at = $7
		WHERE id = $1 AND user_id = $2 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query,
		b.ID, b.UserID, b.Name, b.LimitIDR, b.CategoryID, b.AlertThreshold, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("gagal update anggaran: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrBudgetNotFound
	}
	return nil
}

func (r *postgresBudgetRepository) SoftDelete(ctx context.Context, tenantSchema, id, userID string) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.budgets
		SET is_deleted = true, deleted_at = now(), updated_at = now()
		WHERE id = $1 AND user_id = $2 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("gagal soft delete anggaran: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrBudgetNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanBudgetRow(s scanner) (entity.BudgetWithProgress, error) {
	var b entity.BudgetWithProgress
	var categoryID *uuid.UUID
	var periodType string
	err := s.Scan(
		&b.ID, &b.UserID, &b.SyncID, &b.Name, &b.LimitIDR, &categoryID,
		&periodType, &b.StartDate, &b.EndDate, &b.AlertThreshold,
		&b.IsDeleted, &b.DeletedAt, &b.CreatedAt, &b.UpdatedAt,
		&b.CategoryName, &b.SpentIDR,
	)
	if err != nil {
		return b, err
	}
	b.CategoryID = categoryID
	b.PeriodType = entity.BudgetPeriodType(periodType)
	return b, nil
}
