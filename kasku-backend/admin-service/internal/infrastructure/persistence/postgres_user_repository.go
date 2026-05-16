package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresUserRepository punya akses R/W ke kasku_auth.users.
// Pool yang dipakai harus connect dengan user kasku_auth_svc (atau credentials yang setara).
type postgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository membuat repository read+write untuk kasku_auth.users.
// Mengimplementasikan repository.UserReadRepository + repository.UserWriteRepository.
func NewPostgresUserRepository(pool *pgxpool.Pool) *postgresUserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) List(ctx context.Context, f repository.UserListFilter) ([]entity.UserSummary, int64, error) {
	var (
		conds []string
		args  []any
		i     = 1
	)

	if f.Query != "" {
		conds = append(conds, fmt.Sprintf("(LOWER(email) LIKE LOWER($%d) OR LOWER(username) LIKE LOWER($%d))", i, i))
		args = append(args, "%"+f.Query+"%")
		i++
	}
	if f.IsActive != nil {
		conds = append(conds, fmt.Sprintf("is_active = $%d", i))
		args = append(args, *f.IsActive)
		i++
	}
	if f.EmailVerified != nil {
		conds = append(conds, fmt.Sprintf("email_verified = $%d", i))
		args = append(args, *f.EmailVerified)
		i++
	}
	if f.CreatedFrom != nil {
		conds = append(conds, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, *f.CreatedFrom)
		i++
	}
	if f.CreatedTo != nil {
		conds = append(conds, fmt.Sprintf("created_at < $%d", i))
		args = append(args, *f.CreatedTo)
		i++
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int64
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM public.users %s", where)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal count users: %w", err)
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	listQ := fmt.Sprintf(`
		SELECT id, email, username, is_active, email_verified, created_at, last_login_at
		FROM public.users
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, i, i+1)

	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal query users: %w", err)
	}
	defer rows.Close()

	out := make([]entity.UserSummary, 0, limit)
	for rows.Next() {
		var u entity.UserSummary
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.IsActive, &u.EmailVerified, &u.CreatedAt, &u.LastLoginAt); err != nil {
			return nil, 0, fmt.Errorf("gagal scan user: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterasi users: %w", err)
	}
	return out, total, nil
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.UserSummary, error) {
	const q = `
		SELECT id, email, username, is_active, email_verified, created_at, last_login_at
		FROM public.users
		WHERE id = $1
		LIMIT 1
	`
	var u entity.UserSummary
	err := r.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.Username, &u.IsActive, &u.EmailVerified, &u.CreatedAt, &u.LastLoginAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal get user by id: %w", err)
	}
	return &u, nil
}

func (r *postgresUserRepository) CountTotal(ctx context.Context) (int64, error) {
	var n int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM public.users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("gagal count users: %w", err)
	}
	return n, nil
}

func (r *postgresUserRepository) CountActive(ctx context.Context) (int64, error) {
	var n int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM public.users WHERE is_active = true`).Scan(&n); err != nil {
		return 0, fmt.Errorf("gagal count active users: %w", err)
	}
	return n, nil
}

func (r *postgresUserRepository) CountCreatedSince(ctx context.Context, since time.Time) (int64, error) {
	var n int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM public.users WHERE created_at >= $1`, since).Scan(&n); err != nil {
		return 0, fmt.Errorf("gagal count users since: %w", err)
	}
	return n, nil
}

func (r *postgresUserRepository) SetIsActive(ctx context.Context, userID uuid.UUID, isActive bool) error {
	const q = `UPDATE public.users SET is_active = $2 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, userID, isActive)
	if err != nil {
		return fmt.Errorf("gagal update users.is_active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user tidak ditemukan")
	}
	return nil
}
