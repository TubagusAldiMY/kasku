package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresUserRepository mengimplementasikan repository.UserRepository menggunakan pgxpool.
type postgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository membuat instance postgresUserRepository.
func NewPostgresUserRepository(pool *pgxpool.Pool) repository.UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email, username, password_hash, is_active, email_verified,
		       failed_login_count, locked_until, last_login_at, created_at, updated_at
		FROM public.users
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1
	`
	return r.scanUser(ctx, query, email)
}

func (r *postgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `
		SELECT id, email, username, password_hash, is_active, email_verified,
		       failed_login_count, locked_until, last_login_at, created_at, updated_at
		FROM public.users
		WHERE id = $1
		LIMIT 1
	`
	return r.scanUser(ctx, query, id)
}

func (r *postgresUserRepository) scanUser(ctx context.Context, query string, arg interface{}) (*entity.User, error) {
	row := r.pool.QueryRow(ctx, query, arg)

	var u entity.User
	err := row.Scan(
		&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.IsActive, &u.EmailVerified,
		&u.FailedLoginCount, &u.LockedUntil, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal scan user: %w", err)
	}

	return &u, nil
}

func (r *postgresUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM public.users WHERE LOWER(email) = LOWER($1))`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("gagal cek keberadaan email: %w", err)
	}
	return exists, nil
}

func (r *postgresUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM public.users WHERE LOWER(username) = LOWER($1))`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, username).Scan(&exists); err != nil {
		return false, fmt.Errorf("gagal cek keberadaan username: %w", err)
	}
	return exists, nil
}

func (r *postgresUserRepository) Create(ctx context.Context, u *entity.User) error {
	query := `
		INSERT INTO public.users (id, email, username, password_hash, is_active, email_verified,
		                          failed_login_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID, u.Email, u.Username, u.PasswordHash, u.IsActive, u.EmailVerified,
		u.FailedLoginCount, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert user: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) UpdateLoginSuccess(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE public.users
		SET failed_login_count = 0, last_login_at = $2, locked_until = NULL, updated_at = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, userID, now, now)
	if err != nil {
		return fmt.Errorf("gagal update login success: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) IncrementFailedLoginAndLockIfNeeded(
	ctx context.Context,
	userID uuid.UUID,
	maxAttempts int16,
	lockoutDurationStr string,
) error {
	lockoutDuration, err := time.ParseDuration(lockoutDurationStr)
	if err != nil {
		return fmt.Errorf("durasi lockout tidak valid: %w", err)
	}

	now := time.Now().UTC()
	lockUntil := now.Add(lockoutDuration)

	// Gunakan CASE WHEN di SQL untuk atomisitas — satu round-trip ke DB
	query := `
		UPDATE public.users
		SET
			failed_login_count = LEAST(failed_login_count + 1, $2),
			locked_until = CASE
				WHEN failed_login_count + 1 >= $2 THEN $3
				ELSE locked_until
			END,
			updated_at = $4
		WHERE id = $1
	`
	_, err = r.pool.Exec(ctx, query, userID, maxAttempts, lockUntil, now)
	if err != nil {
		return fmt.Errorf("gagal increment failed login count: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE public.users
		SET is_active = true, email_verified = true, updated_at = $2
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, userID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("gagal update email verified: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error {
	query := `
		UPDATE public.users
		SET password_hash = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, userID, newPasswordHash, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("gagal update password: %w", err)
	}
	return nil
}
