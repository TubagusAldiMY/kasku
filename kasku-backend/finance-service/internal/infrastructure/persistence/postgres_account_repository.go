package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresAccountRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAccountRepository(pool *pgxpool.Pool) repository.FinancialAccountRepository {
	return &postgresAccountRepository{pool: pool}
}

func (r *postgresAccountRepository) CountByUserID(ctx context.Context, tenantSchema, userID string) (int, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return 0, err
	}
	// Schema name divalidasi oleh regex sebelum interpolasi — tidak ada SQL injection risk
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s.financial_accounts WHERE user_id = $1 AND is_deleted = false",
		tenantSchema,
	)
	var count int
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("gagal hitung akun: %w", err)
	}
	return count, nil
}

func (r *postgresAccountRepository) Create(ctx context.Context, tenantSchema string, account *entity.FinancialAccount) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		INSERT INTO %s.financial_accounts
			(id, user_id, name, account_type, balance, initial_balance, currency, color, icon, is_default, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9, false, $10, $11)
	`, tenantSchema)
	_, err := r.pool.Exec(ctx, query,
		account.ID, account.UserID, account.Name, string(account.AccountType),
		account.Balance, account.Currency, account.Color, account.Icon,
		account.IsDefault, account.CreatedAt, account.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert akun: %w", err)
	}
	return nil
}

func (r *postgresAccountRepository) List(ctx context.Context, tenantSchema, userID string) ([]entity.FinancialAccount, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id, user_id, name, account_type, balance, currency, color, icon, is_default, is_deleted, deleted_at, created_at, updated_at
		FROM %s.financial_accounts
		WHERE user_id = $1 AND is_deleted = false
		ORDER BY is_default DESC, created_at ASC
	`, tenantSchema)

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal query akun: %w", err)
	}
	defer rows.Close()

	var accounts []entity.FinancialAccount
	for rows.Next() {
		a := entity.FinancialAccount{}
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Name, &a.AccountType,
			&a.Balance, &a.Currency, &a.Color, &a.Icon,
			&a.IsDefault, &a.IsDeleted, &a.DeletedAt,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan akun: %w", err)
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi baris akun: %w", err)
	}
	return accounts, nil
}

func (r *postgresAccountRepository) GetByID(ctx context.Context, tenantSchema, id, userID string) (*entity.FinancialAccount, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id, user_id, name, account_type, balance, currency, color, icon, is_default, is_deleted, deleted_at, created_at, updated_at
		FROM %s.financial_accounts
		WHERE id = $1 AND user_id = $2 AND is_deleted = false
	`, tenantSchema)

	a := &entity.FinancialAccount{}
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&a.ID, &a.UserID, &a.Name, &a.AccountType,
		&a.Balance, &a.Currency, &a.Color, &a.Icon,
		&a.IsDefault, &a.IsDeleted, &a.DeletedAt,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrAccountNotFound
		}
		return nil, fmt.Errorf("gagal get akun: %w", err)
	}
	return a, nil
}

func (r *postgresAccountRepository) Update(ctx context.Context, tenantSchema string, account *entity.FinancialAccount) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.financial_accounts
		SET name = $3, account_type = $4, color = $5, icon = $6, is_default = $7, updated_at = $8
		WHERE id = $1 AND user_id = $2 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query,
		account.ID, account.UserID, account.Name, string(account.AccountType),
		account.Color, account.Icon, account.IsDefault, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("gagal update akun: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrAccountNotFound
	}
	return nil
}

func (r *postgresAccountRepository) SoftDelete(ctx context.Context, tenantSchema, id, userID string) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.financial_accounts
		SET is_deleted = true, deleted_at = now(), updated_at = now()
		WHERE id = $1 AND user_id = $2 AND is_deleted = false
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("gagal soft delete akun: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrAccountNotFound
	}
	return nil
}

func (r *postgresAccountRepository) GetBalanceHistory(ctx context.Context, tenantSchema, accountID string, limitMonths int) ([]entity.BalanceHistory, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}

	var (
		query string
		args  []interface{}
	)

	// limitMonths == 0 berarti ambil semua history (unlimited)
	if limitMonths > 0 {
		query = fmt.Sprintf(`
			SELECT id, account_id, amount, balance, note, created_at
			FROM %s.balance_history
			WHERE account_id = $1 AND created_at >= now() - ($2 || ' months')::interval
			ORDER BY created_at DESC
		`, tenantSchema)
		args = []interface{}{accountID, limitMonths}
	} else {
		query = fmt.Sprintf(`
			SELECT id, account_id, amount, balance, note, created_at
			FROM %s.balance_history
			WHERE account_id = $1
			ORDER BY created_at DESC
		`, tenantSchema)
		args = []interface{}{accountID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query balance history: %w", err)
	}
	defer rows.Close()

	var history []entity.BalanceHistory
	for rows.Next() {
		h := entity.BalanceHistory{}
		if err := rows.Scan(&h.ID, &h.AccountID, &h.Amount, &h.Balance, &h.Note, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("gagal scan balance history: %w", err)
		}
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi baris balance history: %w", err)
	}
	return history, nil
}
