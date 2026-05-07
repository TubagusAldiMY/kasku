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

// postgresEmailVerificationRepository mengimplementasikan repository.EmailVerificationRepository.
type postgresEmailVerificationRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresEmailVerificationRepository membuat instance postgresEmailVerificationRepository.
func NewPostgresEmailVerificationRepository(pool *pgxpool.Pool) repository.EmailVerificationRepository {
	return &postgresEmailVerificationRepository{pool: pool}
}

func (r *postgresEmailVerificationRepository) Create(ctx context.Context, v *entity.EmailVerification) error {
	query := `
		INSERT INTO public.email_verifications (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, v.ID, v.UserID, v.TokenHash, v.ExpiresAt, v.CreatedAt)
	if err != nil {
		return fmt.Errorf("gagal insert email verification: %w", err)
	}
	return nil
}

func (r *postgresEmailVerificationRepository) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.EmailVerification, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, verified_at, created_at
		FROM public.email_verifications
		WHERE token_hash = $1
		  AND verified_at IS NULL
		  AND expires_at > now()
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)

	var ev entity.EmailVerification
	err := row.Scan(&ev.ID, &ev.UserID, &ev.TokenHash, &ev.ExpiresAt, &ev.VerifiedAt, &ev.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal scan email verification: %w", err)
	}

	return &ev, nil
}

func (r *postgresEmailVerificationRepository) MarkAsVerified(ctx context.Context, verificationID uuid.UUID) error {
	query := `
		UPDATE public.email_verifications
		SET verified_at = $2
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, verificationID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("gagal mark email verification as verified: %w", err)
	}
	return nil
}

func (r *postgresEmailVerificationRepository) InvalidateAllActiveByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE public.email_verifications
		SET verified_at = $2
		WHERE user_id = $1 AND verified_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, userID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("gagal invalidate email verifications: %w", err)
	}
	return nil
}
