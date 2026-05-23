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

type postgresTransactionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTransactionRepository(pool *pgxpool.Pool) repository.TransactionRepository {
	return &postgresTransactionRepository{pool: pool}
}

func (r *postgresTransactionRepository) CountMonthly(ctx context.Context, tenantSchema, userID string, month time.Time) (int, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return 0, err
	}
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.transactions t
		JOIN %s.financial_accounts a ON t.account_id = a.id
		WHERE a.user_id = $1 AND t.is_deleted = false
		  AND t.transaction_date >= $2 AND t.transaction_date < $3
	`, tenantSchema, tenantSchema)
	var count int
	if err := r.pool.QueryRow(ctx, query, userID, start, end).Scan(&count); err != nil {
		return 0, fmt.Errorf("gagal hitung transaksi bulanan: %w", err)
	}
	return count, nil
}

func (r *postgresTransactionRepository) Create(ctx context.Context, tenantSchema string, tx *entity.Transaction) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	if tx.TransactionType != entity.TransactionExpense {
		tx.BudgetID = nil
	}

	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gagal mulai transaksi DB: %w", err)
	}
	defer dbTx.Rollback(ctx) //nolint:errcheck

	// Validasi saldo mencukupi untuk TRANSFER — dilakukan di dalam dbTx dengan FOR UPDATE
	// agar atomik terhadap concurrent request (mencegah race condition TOCTOU).
	if tx.TransactionType == entity.TransactionTransfer {
		var currentBalance int64
		balanceQ := fmt.Sprintf(
			"SELECT balance FROM %s.financial_accounts WHERE id = $1 AND is_deleted = false FOR UPDATE",
			tenantSchema,
		)
		err := dbTx.QueryRow(ctx, balanceQ, tx.AccountID).Scan(&currentBalance)
		if errors.Is(err, pgx.ErrNoRows) {
			return domainerrors.ErrAccountNotFound
		}
		if err != nil {
			return fmt.Errorf("gagal membaca saldo rekening asal: %w", err)
		}
		if tx.AmountIDR > currentBalance {
			return domainerrors.ErrInsufficientBalance
		}
	}
	if err := validateBudgetForAccount(ctx, dbTx, tenantSchema, tx.AccountID, tx.BudgetID); err != nil {
		return err
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO %s.transactions
			(id, sync_id, account_id, category_id, budget_id, transaction_type, amount_idr, transaction_date, notes, to_account_id, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false, $11, $12)
		ON CONFLICT (sync_id) DO NOTHING
	`, tenantSchema)
	tag, err := dbTx.Exec(ctx, insertQuery,
		tx.ID, tx.SyncID, tx.AccountID, tx.CategoryID, tx.BudgetID,
		string(tx.TransactionType), tx.AmountIDR, tx.TransactionDate,
		tx.Notes, tx.ToAccountID, tx.CreatedAt, tx.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert transaksi: %w", err)
	}
	// ON CONFLICT (sync_id) DO NOTHING — duplicate sync_id dianggap idempotent
	if tag.RowsAffected() == 0 {
		return dbTx.Commit(ctx)
	}

	if err := RecalculateAccountBalance(ctx, dbTx, tenantSchema, tx.AccountID); err != nil {
		return err
	}
	if tx.ToAccountID != nil {
		if err := RecalculateAccountBalance(ctx, dbTx, tenantSchema, *tx.ToAccountID); err != nil {
			return err
		}
	}

	return dbTx.Commit(ctx)
}

// RecalculateAccountBalance recomputes an account's balance from scratch:
// balance = initial_balance + aggregate of all non-deleted transactions.
// This is idempotent and always correct regardless of prior balance state.
func RecalculateAccountBalance(ctx context.Context, dbTx pgx.Tx, tenantSchema string, accountID uuid.UUID) error {
	q := fmt.Sprintf(`
		UPDATE %s.financial_accounts fa SET
			balance = fa.initial_balance
				+ COALESCE((
					SELECT SUM(CASE
						WHEN t.transaction_type = 'INCOME'   THEN  t.amount_idr
						WHEN t.transaction_type = 'EXPENSE'  THEN -t.amount_idr
						WHEN t.transaction_type = 'TRANSFER' THEN -t.amount_idr
						ELSE 0 END)
					FROM %s.transactions t
					WHERE t.account_id = $1 AND t.is_deleted = false
				), 0)
				+ COALESCE((
					SELECT SUM(t.amount_idr)
					FROM %s.transactions t
					WHERE t.to_account_id = $1 AND t.is_deleted = false
					  AND t.transaction_type = 'TRANSFER'
				), 0),
			updated_at = now()
		WHERE fa.id = $1 AND fa.is_deleted = false
	`, tenantSchema, tenantSchema, tenantSchema)
	if _, err := dbTx.Exec(ctx, q, accountID); err != nil {
		return fmt.Errorf("gagal recalculate saldo akun: %w", err)
	}
	return nil
}

func recalculateAccounts(ctx context.Context, dbTx pgx.Tx, tenantSchema string, accountIDs []uuid.UUID) error {
	seen := make(map[uuid.UUID]struct{}, len(accountIDs))
	for _, id := range accountIDs {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if err := RecalculateAccountBalance(ctx, dbTx, tenantSchema, id); err != nil {
			return err
		}
	}
	return nil
}

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func ValidateAccountForUser(ctx context.Context, q QueryRower, tenantSchema, userID string, accountID uuid.UUID) error {
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %s.financial_accounts
			WHERE id = $1 AND user_id = $2::uuid AND is_deleted = false
		)
	`, tenantSchema)
	var exists bool
	if err := q.QueryRow(ctx, query, accountID, userID).Scan(&exists); err != nil {
		return fmt.Errorf("gagal validasi rekening: %w", err)
	}
	if !exists {
		return domainerrors.ErrAccountNotFound
	}
	return nil
}

func validateBudgetForAccount(ctx context.Context, q QueryRower, tenantSchema string, accountID uuid.UUID, budgetID *uuid.UUID) error {
	if budgetID == nil {
		return nil
	}
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM %s.budgets b
			JOIN %s.financial_accounts a ON a.user_id = b.user_id
			WHERE b.id = $1 AND a.id = $2
			  AND b.is_deleted = false AND a.is_deleted = false
		)
	`, tenantSchema, tenantSchema)
	var exists bool
	if err := q.QueryRow(ctx, query, *budgetID, accountID).Scan(&exists); err != nil {
		return fmt.Errorf("gagal validasi anggaran: %w", err)
	}
	if !exists {
		return domainerrors.ErrBudgetNotFound
	}
	return nil
}

func ValidateBudgetForUser(ctx context.Context, q QueryRower, tenantSchema, userID string, budgetID *uuid.UUID) error {
	if budgetID == nil {
		return nil
	}
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %s.budgets
			WHERE id = $1 AND user_id = $2::uuid AND is_deleted = false
		)
	`, tenantSchema)
	var exists bool
	if err := q.QueryRow(ctx, query, *budgetID, userID).Scan(&exists); err != nil {
		return fmt.Errorf("gagal validasi anggaran: %w", err)
	}
	if !exists {
		return domainerrors.ErrBudgetNotFound
	}
	return nil
}

func (r *postgresTransactionRepository) List(ctx context.Context, tenantSchema, userID string, from, to time.Time) ([]entity.Transaction, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT t.id, t.sync_id, t.account_id, t.category_id, t.budget_id, t.transaction_type,
		       t.amount_idr, t.transaction_date, t.notes, t.to_account_id,
		       t.is_deleted, t.deleted_at, t.created_at, t.updated_at
		FROM %s.transactions t
		JOIN %s.financial_accounts a ON t.account_id = a.id
		WHERE a.user_id = $1 AND t.is_deleted = false
		  AND t.transaction_date >= $2 AND t.transaction_date <= $3
		ORDER BY t.transaction_date DESC, t.created_at DESC
	`, tenantSchema, tenantSchema)
	rows, err := r.pool.Query(ctx, query, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("gagal query transaksi: %w", err)
	}
	defer rows.Close()

	var txs []entity.Transaction
	for rows.Next() {
		var t entity.Transaction
		if err := rows.Scan(
			&t.ID, &t.SyncID, &t.AccountID, &t.CategoryID, &t.BudgetID, &t.TransactionType,
			&t.AmountIDR, &t.TransactionDate, &t.Notes, &t.ToAccountID,
			&t.IsDeleted, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan transaksi: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

func (r *postgresTransactionRepository) GetByID(ctx context.Context, tenantSchema, id, userID string) (*entity.Transaction, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT t.id, t.sync_id, t.account_id, t.category_id, t.budget_id, t.transaction_type,
		       t.amount_idr, t.transaction_date, t.notes, t.to_account_id,
		       t.is_deleted, t.deleted_at, t.created_at, t.updated_at
		FROM %s.transactions t
		JOIN %s.financial_accounts a ON t.account_id = a.id
		WHERE t.id = $1 AND a.user_id = $2 AND t.is_deleted = false
	`, tenantSchema, tenantSchema)
	t := &entity.Transaction{}
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&t.ID, &t.SyncID, &t.AccountID, &t.CategoryID, &t.BudgetID, &t.TransactionType,
		&t.AmountIDR, &t.TransactionDate, &t.Notes, &t.ToAccountID,
		&t.IsDeleted, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrTransactionNotFound
		}
		return nil, fmt.Errorf("gagal get transaksi: %w", err)
	}
	return t, nil
}

func (r *postgresTransactionRepository) Update(ctx context.Context, tenantSchema, userID string, tx *entity.Transaction) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	if tx.TransactionType != entity.TransactionExpense {
		tx.BudgetID = nil
	}

	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gagal mulai transaksi DB: %w", err)
	}
	defer dbTx.Rollback(ctx) //nolint:errcheck

	selectQuery := fmt.Sprintf(`
		SELECT account_id, to_account_id, transaction_type, amount_idr
		FROM %s.transactions
		WHERE id = $1 AND is_deleted = false
		  AND account_id IN (SELECT id FROM %s.financial_accounts WHERE user_id = $2)
	`, tenantSchema, tenantSchema)
	var oldAccountID uuid.UUID
	var oldToAccountID *uuid.UUID
	var oldType entity.TransactionType
	var oldAmount int64
	err = dbTx.QueryRow(ctx, selectQuery, tx.ID, userID).Scan(&oldAccountID, &oldToAccountID, &oldType, &oldAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainerrors.ErrTransactionNotFound
	}
	if err != nil {
		return fmt.Errorf("gagal ambil transaksi lama: %w", err)
	}

	if err := ValidateAccountForUser(ctx, dbTx, tenantSchema, userID, tx.AccountID); err != nil {
		return err
	}
	if tx.TransactionType == entity.TransactionTransfer {
		if tx.ToAccountID == nil || *tx.ToAccountID == tx.AccountID {
			return fmt.Errorf("%w: rekening tujuan transfer tidak valid", domainerrors.ErrInvalidInput)
		}
		if err := ValidateAccountForUser(ctx, dbTx, tenantSchema, userID, *tx.ToAccountID); err != nil {
			return err
		}
	}

	if tx.TransactionType == entity.TransactionTransfer {
		var currentBalance int64
		balanceQ := fmt.Sprintf(
			"SELECT balance FROM %s.financial_accounts WHERE id = $1 AND user_id = $2::uuid AND is_deleted = false FOR UPDATE",
			tenantSchema,
		)
		err := dbTx.QueryRow(ctx, balanceQ, tx.AccountID, userID).Scan(&currentBalance)
		if errors.Is(err, pgx.ErrNoRows) {
			return domainerrors.ErrAccountNotFound
		}
		if err != nil {
			return fmt.Errorf("gagal membaca saldo rekening asal: %w", err)
		}
		available := currentBalance
		if oldAccountID == tx.AccountID && (oldType == entity.TransactionExpense || oldType == entity.TransactionTransfer) {
			available += oldAmount
		}
		if oldToAccountID != nil && *oldToAccountID == tx.AccountID && oldType == entity.TransactionTransfer {
			available -= oldAmount
		}
		if tx.AmountIDR > available {
			return domainerrors.ErrInsufficientBalance
		}
	}

	if err := validateBudgetForAccount(ctx, dbTx, tenantSchema, tx.AccountID, tx.BudgetID); err != nil {
		return err
	}

	query := fmt.Sprintf(`
		UPDATE %s.transactions
		SET account_id = $3, category_id = $4, budget_id = $5, transaction_type = $6,
		    amount_idr = $7, transaction_date = $8, notes = $9, to_account_id = $10,
		    updated_at = now()
		WHERE id = $1 AND is_deleted = false
		  AND account_id IN (SELECT id FROM %s.financial_accounts WHERE user_id = $2)
	`, tenantSchema, tenantSchema)
	result, err := dbTx.Exec(ctx, query,
		tx.ID, userID, tx.AccountID, tx.CategoryID, tx.BudgetID, string(tx.TransactionType),
		tx.AmountIDR, tx.TransactionDate, tx.Notes, tx.ToAccountID,
	)
	if err != nil {
		return fmt.Errorf("gagal update transaksi: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrTransactionNotFound
	}

	affected := []uuid.UUID{oldAccountID, tx.AccountID}
	if oldToAccountID != nil {
		affected = append(affected, *oldToAccountID)
	}
	if tx.ToAccountID != nil {
		affected = append(affected, *tx.ToAccountID)
	}
	if err := recalculateAccounts(ctx, dbTx, tenantSchema, affected); err != nil {
		return err
	}
	return dbTx.Commit(ctx)
}

func (r *postgresTransactionRepository) SoftDelete(ctx context.Context, tenantSchema, id, userID string) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}

	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gagal mulai transaksi DB: %w", err)
	}
	defer dbTx.Rollback(ctx) //nolint:errcheck

	// Ambil account_id dan to_account_id sekaligus validasi kepemilikan user
	selectQuery := fmt.Sprintf(`
		SELECT account_id, to_account_id
		FROM %s.transactions
		WHERE id = $1 AND is_deleted = false
		  AND account_id IN (SELECT id FROM %s.financial_accounts WHERE user_id = $2)
	`, tenantSchema, tenantSchema)

	var accountID uuid.UUID
	var toAccountID *uuid.UUID
	err = dbTx.QueryRow(ctx, selectQuery, id, userID).Scan(&accountID, &toAccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainerrors.ErrTransactionNotFound
	}
	if err != nil {
		return fmt.Errorf("gagal ambil detail transaksi: %w", err)
	}

	deleteQuery := fmt.Sprintf(`
		UPDATE %s.transactions SET is_deleted = true, deleted_at = now(), updated_at = now()
		WHERE id = $1 AND is_deleted = false
	`, tenantSchema)
	if _, err := dbTx.Exec(ctx, deleteQuery, id); err != nil {
		return fmt.Errorf("gagal soft delete transaksi: %w", err)
	}

	if err := RecalculateAccountBalance(ctx, dbTx, tenantSchema, accountID); err != nil {
		return err
	}
	if toAccountID != nil {
		if err := RecalculateAccountBalance(ctx, dbTx, tenantSchema, *toAccountID); err != nil {
			return err
		}
	}

	return dbTx.Commit(ctx)
}

func (r *postgresTransactionRepository) GetSummary(ctx context.Context, tenantSchema, userID string, from, to time.Time) (*entity.TransactionSummary, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT
		    COALESCE(SUM(CASE WHEN t.transaction_type = 'INCOME' THEN t.amount_idr ELSE 0 END), 0) AS total_income,
		    COALESCE(SUM(CASE WHEN t.transaction_type = 'EXPENSE' THEN t.amount_idr ELSE 0 END), 0) AS total_expense
		FROM %s.transactions t
		JOIN %s.financial_accounts a ON t.account_id = a.id
		WHERE a.user_id = $1 AND t.is_deleted = false
		  AND t.transaction_date >= $2 AND t.transaction_date <= $3
	`, tenantSchema, tenantSchema)
	s := &entity.TransactionSummary{}
	if err := r.pool.QueryRow(ctx, query, userID, from, to).Scan(&s.TotalIncome, &s.TotalExpense); err != nil {
		return nil, fmt.Errorf("gagal ambil summary: %w", err)
	}
	s.NetAmount = s.TotalIncome - s.TotalExpense
	return s, nil
}

func (r *postgresTransactionRepository) ListForExport(ctx context.Context, tenantSchema, userID string, from, to *time.Time) ([]entity.Transaction, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT t.id, t.sync_id, t.account_id, t.category_id, t.budget_id, t.transaction_type,
		       t.amount_idr, t.transaction_date, t.notes, t.to_account_id,
		       t.is_deleted, t.deleted_at, t.created_at, t.updated_at
		FROM %s.transactions t
		JOIN %s.financial_accounts a ON t.account_id = a.id
		WHERE a.user_id = $1 AND t.is_deleted = false
		ORDER BY t.transaction_date DESC
	`, tenantSchema, tenantSchema)
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal query untuk export: %w", err)
	}
	defer rows.Close()
	var txs []entity.Transaction
	for rows.Next() {
		var t entity.Transaction
		if err := rows.Scan(
			&t.ID, &t.SyncID, &t.AccountID, &t.CategoryID, &t.BudgetID, &t.TransactionType,
			&t.AmountIDR, &t.TransactionDate, &t.Notes, &t.ToAccountID,
			&t.IsDeleted, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan export row: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}
