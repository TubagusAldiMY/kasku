package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresAdminUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAdminUserRepository membuat repository admin_users (kasku_admin DB).
func NewPostgresAdminUserRepository(pool *pgxpool.Pool) repository.AdminUserRepository {
	return &postgresAdminUserRepository{pool: pool}
}

func (r *postgresAdminUserRepository) FindByUsername(ctx context.Context, username string) (*entity.AdminUser, error) {
	const q = `
		SELECT id, username, password_hash, role, is_active, last_login_at, created_at, updated_at
		FROM public.admin_users
		WHERE LOWER(username) = LOWER($1)
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, q, username)
	return scanAdminUser(row)
}

func (r *postgresAdminUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.AdminUser, error) {
	const q = `
		SELECT id, username, password_hash, role, is_active, last_login_at, created_at, updated_at
		FROM public.admin_users
		WHERE id = $1
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, q, id)
	return scanAdminUser(row)
}

func (r *postgresAdminUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	const q = `UPDATE public.admin_users SET last_login_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id, at)
	if err != nil {
		return fmt.Errorf("gagal update last_login_at admin: %w", err)
	}
	return nil
}

func (r *postgresAdminUserRepository) Count(ctx context.Context) (int64, error) {
	const q = `SELECT COUNT(*) FROM public.admin_users`
	var n int64
	if err := r.pool.QueryRow(ctx, q).Scan(&n); err != nil {
		return 0, fmt.Errorf("gagal count admin_users: %w", err)
	}
	return n, nil
}

func (r *postgresAdminUserRepository) CreateBootstrap(ctx context.Context, admin *entity.AdminUser) error {
	const q = `
		INSERT INTO public.admin_users (id, username, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`
	_, err := r.pool.Exec(ctx, q, admin.ID, admin.Username, admin.PasswordHash, string(admin.Role), admin.IsActive)
	if err != nil {
		return fmt.Errorf("gagal bootstrap admin: %w", err)
	}
	return nil
}

func scanAdminUser(row pgx.Row) (*entity.AdminUser, error) {
	var a entity.AdminUser
	var roleStr string
	err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &roleStr, &a.IsActive, &a.LastLoginAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal scan admin_user: %w", err)
	}
	a.Role = entity.AdminRole(roleStr)
	return &a, nil
}
