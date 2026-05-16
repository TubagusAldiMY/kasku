package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserProfileRepository mendefinisikan operasi terhadap kasku_user.user_profiles.
// Tabel ini dimiliki user-service.
type UserProfileRepository interface {
	EnsureUserProfile(ctx context.Context, userID, email, username string) error
	GetUserProfile(ctx context.Context, userID string) (*UserProfileRecord, error)
	UpdateUserProfile(ctx context.Context, userID, username, displayName string) (*UserProfileRecord, error)
}

type UserProfileRecord struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName *string   `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type postgresUserProfileRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserProfileRepository(pool *pgxpool.Pool) UserProfileRepository {
	return &postgresUserProfileRepository{pool: pool}
}

func (r *postgresUserProfileRepository) EnsureUserProfile(ctx context.Context, userID, email, username string) error {
	query := `
		INSERT INTO public.user_profiles (user_id, email, username, display_name)
		VALUES ($1::uuid, $2, $3, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET email = EXCLUDED.email,
		    username = EXCLUDED.username
	`
	_, err := r.pool.Exec(ctx, query, userID, email, username)
	if err != nil {
		return fmt.Errorf("gagal upsert user profile untuk user %s: %w", userID, err)
	}
	return nil
}

func (r *postgresUserProfileRepository) GetUserProfile(ctx context.Context, userID string) (*UserProfileRecord, error) {
	query := `
		SELECT user_id::text, email, username, display_name, created_at, updated_at
		FROM public.user_profiles
		WHERE user_id = $1::uuid
	`
	row := &UserProfileRecord{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&row.UserID, &row.Email, &row.Username, &row.DisplayName, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal get user profile untuk user %s: %w", userID, err)
	}
	return row, nil
}

func (r *postgresUserProfileRepository) UpdateUserProfile(ctx context.Context, userID, username, displayName string) (*UserProfileRecord, error) {
	query := `
		UPDATE public.user_profiles
		SET username = COALESCE(NULLIF($2, ''), username),
		    display_name = COALESCE(NULLIF($3, ''), display_name)
		WHERE user_id = $1::uuid
		RETURNING user_id::text, email, username, display_name, created_at, updated_at
	`
	row := &UserProfileRecord{}
	err := r.pool.QueryRow(ctx, query, userID, username, displayName).Scan(
		&row.UserID, &row.Email, &row.Username, &row.DisplayName, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal update user profile untuk user %s: %w", userID, err)
	}
	return row, nil
}
