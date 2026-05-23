package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresCategoryRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCategoryRepository(pool *pgxpool.Pool) repository.CategoryRepository {
	return &postgresCategoryRepository{pool: pool}
}

func (r *postgresCategoryRepository) List(ctx context.Context, tenantSchema string) ([]entity.Category, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id, name, icon, color, category_type, is_default, is_deleted, deleted_at, created_at, updated_at
		FROM %s.categories
		WHERE is_deleted = false
		ORDER BY is_default DESC, name ASC
	`, tenantSchema)
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query kategori: %w", err)
	}
	defer rows.Close()
	var cats []entity.Category
	for rows.Next() {
		c := entity.Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.Color, &c.CategoryType,
			&c.IsDefault, &c.IsDeleted, &c.DeletedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("gagal scan kategori: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (r *postgresCategoryRepository) GetByID(ctx context.Context, tenantSchema, id string) (*entity.Category, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id, name, icon, color, category_type, is_default, is_deleted, deleted_at, created_at, updated_at
		FROM %s.categories WHERE id = $1 AND is_deleted = false
	`, tenantSchema)
	c := &entity.Category{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.Icon, &c.Color,
		&c.CategoryType, &c.IsDefault, &c.IsDeleted, &c.DeletedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal get kategori: %w", err)
	}
	return c, nil
}

func (r *postgresCategoryRepository) Create(ctx context.Context, tenantSchema string, cat *entity.Category) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		INSERT INTO %s.categories (id, name, icon, color, category_type, is_default, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, false, false, $6, $7)
	`, tenantSchema)
	_, err := r.pool.Exec(ctx, query, cat.ID, cat.Name, cat.Icon, cat.Color,
		string(cat.CategoryType), cat.CreatedAt, cat.UpdatedAt)
	return err
}

func (r *postgresCategoryRepository) Update(ctx context.Context, tenantSchema string, cat *entity.Category) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.categories SET name = $2, icon = $3, color = $4, category_type = $5, updated_at = $6
		WHERE id = $1 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query, cat.ID, cat.Name, cat.Icon,
		cat.Color, string(cat.CategoryType), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("gagal update kategori: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrCategoryNotFound
	}
	return nil
}

func (r *postgresCategoryRepository) SoftDelete(ctx context.Context, tenantSchema, id string) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.categories SET is_deleted = true, deleted_at = now(), updated_at = now()
		WHERE id = $1 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("gagal soft delete kategori: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrCategoryNotFound
	}
	return nil
}

func (r *postgresCategoryRepository) HasActiveTransactions(ctx context.Context, tenantSchema, categoryID string) (bool, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return false, err
	}
	query := fmt.Sprintf(`
		SELECT EXISTS(SELECT 1 FROM %s.transactions WHERE category_id = $1 AND is_deleted = false)
	`, tenantSchema)
	var exists bool
	if err := r.pool.QueryRow(ctx, query, categoryID).Scan(&exists); err != nil {
		return false, fmt.Errorf("gagal cek transaksi aktif: %w", err)
	}
	return exists, nil
}
