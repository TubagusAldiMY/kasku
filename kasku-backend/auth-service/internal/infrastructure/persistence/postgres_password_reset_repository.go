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

// postgresPasswordResetRepository mengimplementasikan repository.PasswordResetRepository
// dan repository.TransactionalResetPasswordRepository.
type postgresPasswordResetRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPasswordResetRepository membuat instance postgresPasswordResetRepository.
func NewPostgresPasswordResetRepository(pool *pgxpool.Pool) (repository.PasswordResetRepository, repository.TransactionalResetPasswordRepository) {
	r := &postgresPasswordResetRepository{pool: pool}
	return r, r
}

func (r *postgresPasswordResetRepository) Create(ctx context.Context, token *entity.PasswordResetToken) error {
	query := `
		INSERT INTO public.password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("gagal insert password reset token: %w", err)
	}
	return nil
}

func (r *postgresPasswordResetRepository) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM public.password_reset_tokens
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > now()
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)

	var pt entity.PasswordResetToken
	err := row.Scan(&pt.ID, &pt.UserID, &pt.TokenHash, &pt.ExpiresAt, &pt.UsedAt, &pt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal scan password reset token: %w", err)
	}

	return &pt, nil
}

func (r *postgresPasswordResetRepository) MarkAsUsed(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE public.password_reset_tokens
		SET used_at = $2
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, tokenID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("gagal mark password reset token as used: %w", err)
	}
	return nil
}

// ExecuteResetPasswordTx menjalankan 3 operasi dalam satu transaksi database:
// 1. Update password_hash di users
// 2. Tandai token reset sebagai sudah digunakan
// 3. Revoke semua refresh token aktif milik user (paksa logout dari semua device)
func (r *postgresPasswordResetRepository) ExecuteResetPasswordTx(
	ctx context.Context,
	userID uuid.UUID,
	newPasswordHash string,
	tokenID uuid.UUID,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi reset password: %w", err)
	}
	defer func() {
		// Rollback diabaikan jika commit sudah berhasil (pgx mengembalikan ErrTxClosed)
		_ = tx.Rollback(ctx)
	}()

	now := time.Now().UTC()

	// Langkah 1: Update password
	if _, err := tx.Exec(ctx,
		`UPDATE public.users SET password_hash = $2, updated_at = $3 WHERE id = $1`,
		userID, newPasswordHash, now,
	); err != nil {
		return fmt.Errorf("gagal update password dalam transaksi: %w", err)
	}

	// Langkah 2: Tandai token reset sebagai used
	if _, err := tx.Exec(ctx,
		`UPDATE public.password_reset_tokens SET used_at = $2 WHERE id = $1`,
		tokenID, now,
	); err != nil {
		return fmt.Errorf("gagal mark reset token dalam transaksi: %w", err)
	}

	// Langkah 3: Revoke semua refresh token aktif (paksa logout dari semua device)
	if _, err := tx.Exec(ctx,
		`UPDATE public.refresh_tokens SET is_revoked = true, revoked_at = $2 WHERE user_id = $1 AND is_revoked = false`,
		userID, now,
	); err != nil {
		return fmt.Errorf("gagal revoke refresh tokens dalam transaksi: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("gagal commit transaksi reset password: %w", err)
	}

	return nil
}

// Pastikan interface terpenuhi pada compile time
var _ repository.PasswordResetRepository = (*postgresPasswordResetRepository)(nil)
var _ repository.TransactionalResetPasswordRepository = (*postgresPasswordResetRepository)(nil)
