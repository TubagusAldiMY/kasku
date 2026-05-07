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

// postgresRefreshTokenRepository mengimplementasikan repository.RefreshTokenRepository.
type postgresRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRefreshTokenRepository membuat instance postgresRefreshTokenRepository.
func NewPostgresRefreshTokenRepository(pool *pgxpool.Pool) repository.RefreshTokenRepository {
	return &postgresRefreshTokenRepository{pool: pool}
}

func (r *postgresRefreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	query := `
		INSERT INTO public.refresh_tokens (id, user_id, token_hash, user_agent, ip_address, expires_at, is_revoked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		token.ID, token.UserID, token.TokenHash,
		token.UserAgent, token.IPAddress,
		token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert refresh token: %w", err)
	}
	return nil
}

func (r *postgresRefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	// Cast ip_address INET → text agar dapat di-scan ke *string
	query := `
		SELECT id, user_id, token_hash, user_agent, ip_address::text, expires_at, is_revoked, revoked_at, created_at
		FROM public.refresh_tokens
		WHERE token_hash = $1
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)

	var rt entity.RefreshToken
	err := row.Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash,
		&rt.UserAgent, &rt.IPAddress,
		&rt.ExpiresAt, &rt.IsRevoked, &rt.RevokedAt, &rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal scan refresh token: %w", err)
	}

	return &rt, nil
}

func (r *postgresRefreshTokenRepository) RevokeByID(ctx context.Context, tokenID uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE public.refresh_tokens
		SET is_revoked = true, revoked_at = $2
		WHERE id = $1 AND is_revoked = false
	`
	_, err := r.pool.Exec(ctx, query, tokenID, now)
	if err != nil {
		return fmt.Errorf("gagal revoke refresh token by ID: %w", err)
	}
	return nil
}

func (r *postgresRefreshTokenRepository) RevokeAllActiveByUserID(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE public.refresh_tokens
		SET is_revoked = true, revoked_at = $2
		WHERE user_id = $1 AND is_revoked = false
	`
	_, err := r.pool.Exec(ctx, query, userID, now)
	if err != nil {
		return fmt.Errorf("gagal revoke semua refresh token untuk user: %w", err)
	}
	return nil
}

// RevokeAllByUserIDInTx diimplementasikan oleh TransactionalPasswordResetRepository.
// Method ini ada karena interface mendefinisikannya, tapi penggunaannya via transaksi pgx.
func (r *postgresRefreshTokenRepository) RevokeAllByUserIDInTx(ctx context.Context, tx interface{}, userID uuid.UUID) error {
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return fmt.Errorf("transaksi tidak valid: bukan pgx.Tx")
	}

	now := time.Now().UTC()
	query := `
		UPDATE public.refresh_tokens
		SET is_revoked = true, revoked_at = $2
		WHERE user_id = $1 AND is_revoked = false
	`
	_, err := pgxTx.Exec(ctx, query, userID, now)
	if err != nil {
		return fmt.Errorf("gagal revoke refresh token dalam transaksi: %w", err)
	}
	return nil
}
