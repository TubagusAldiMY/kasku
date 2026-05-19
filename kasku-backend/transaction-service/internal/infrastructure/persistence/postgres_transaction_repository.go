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

	dbTx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gagal mulai transaksi DB: %w", err)
	}
	defer dbTx.Rollback(ctx) //nolint:errcheck

	insertQuery := fmt.Sprintf(`
		INSERT INTO %s.transactions
			(id, sync_id, account_id, category_id, transaction_type, amount_idr, transaction_date, notes, to_account_id, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, $10, $11)
		ON CONFLICT (sync_id) DO NOTHING
	`, tenantSchema)
	tag, err := dbTx.Exec(ctx, insertQuery,
		tx.ID, tx.SyncID, tx.AccountID, tx.CategoryID,
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

	if err := AdjustBalance(ctx, dbTx, tenantSchema, tx.AccountID, tx.ToAccountID, tx.TransactionType, tx.AmountIDR); err != nil {
		return err
	}

	return dbTx.Commit(ctx)
}

// AdjustBalance mengubah saldo akun sesuai jenis transaksi.
// INCOME: source +amount; EXPENSE: source -amount; TRANSFER: source -amount, dest +amount.
func AdjustBalance(ctx context.Context, dbTx pgx.Tx, tenantSchema string, accountID uuid.UUID, toAccountID *uuid.UUID, txType entity.TransactionType, amount int64) error {
	balanceQuery := fmt.Sprintf(`
		UPDATE %s.financial_accounts SET balance = balance + $1, updated_at = now()
		WHERE id = $2 AND is_deleted = false
	`, tenantSchema)

	var sourceDelta int64
	switch txType {
	case entity.TransactionIncome:
		sourceDelta = amount
	case entity.TransactionExpense:
		sourceDelta = -amount
	case entity.TransactionTransfer:
		sourceDelta = -amount
	}

	if _, err := dbTx.Exec(ctx, balanceQuery, sourceDelta, accountID); err != nil {
		return fmt.Errorf("gagal update saldo sumber: %w", err)
	}

	if txType == entity.TransactionTransfer && toAccountID != nil {
		if _, err := dbTx.Exec(ctx, balanceQuery, amount, *toAccountID); err != nil {
			return fmt.Errorf("gagal update saldo tujuan: %w", err)
		}
	}
	return nil
}

func (r *postgresTransactionRepository) List(ctx context.Context, tenantSchema, userID string, from, to time.Time) ([]entity.Transaction, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT t.id, t.sync_id, t.account_id, t.category_id, t.transaction_type,
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
			&t.ID, &t.SyncID, &t.AccountID, &t.CategoryID, &t.TransactionType,
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
		SELECT t.id, t.sync_id, t.account_id, t.category_id, t.transaction_type,
		       t.amount_idr, t.transaction_date, t.notes, t.to_account_id,
		       t.is_deleted, t.deleted_at, t.created_at, t.updated_at
		FROM %s.transactions t
		JOIN %s.financial_accounts a ON t.account_id = a.id
		WHERE t.id = $1 AND a.user_id = $2 AND t.is_deleted = false
	`, tenantSchema, tenantSchema)
	t := &entity.Transaction{}
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&t.ID, &t.SyncID, &t.AccountID, &t.CategoryID, &t.TransactionType,
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

func (r *postgresTransactionRepository) Update(ctx context.Context, tenantSchema string, tx *entity.Transaction) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.transactions
		SET category_id = $3, transaction_type = $4, amount_idr = $5,
		    transaction_date = $6, notes = $7, to_account_id = $8, updated_at = now()
		WHERE id = $1 AND is_deleted = false
		  AND account_id IN (SELECT id FROM %s.financial_accounts WHERE user_id = $2)
	`, tenantSchema, tenantSchema)
	result, err := r.pool.Exec(ctx, query,
		tx.ID, tx.AccountID, tx.CategoryID, string(tx.TransactionType),
		tx.AmountIDR, tx.TransactionDate, tx.Notes, tx.ToAccountID,
	)
	if err != nil {
		return fmt.Errorf("gagal update transaksi: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrTransactionNotFound
	}
	return nil
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

	// Ambil detail transaksi sekaligus validasi kepemilikan user
	selectQuery := fmt.Sprintf(`
		SELECT transaction_type, amount_idr, account_id, to_account_id
		FROM %s.transactions
		WHERE id = $1 AND is_deleted = false
		  AND account_id IN (SELECT id FROM %s.financial_accounts WHERE user_id = $2)
	`, tenantSchema, tenantSchema)

	var txType entity.TransactionType
	var amount int64
	var accountID uuid.UUID
	var toAccountID *uuid.UUID
	err = dbTx.QueryRow(ctx, selectQuery, id, userID).Scan(&txType, &amount, &accountID, &toAccountID)
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

	// Balik delta saldo (kebalikan dari Create)
	var reversedType entity.TransactionType
	switch txType {
	case entity.TransactionIncome:
		reversedType = entity.TransactionExpense
	case entity.TransactionExpense:
		reversedType = entity.TransactionIncome
	case entity.TransactionTransfer:
		// Untuk transfer: source mendapat kembali, dest dikurangi kembali
		reversedType = entity.TransactionIncome // source +amount
	}
	if err := AdjustBalance(ctx, dbTx, tenantSchema, accountID, toAccountID, reversedType, amount); err != nil {
		return err
	}
	// Khusus TRANSFER: dest dikurangi kembali (adjustBalance tadi menambah dest karena reversedType=INCOME+TRANSFER)
	// Kita perlu override: dest -amount. Gunakan adjustBalance dengan type EXPENSE tanpa to_account.
	if txType == entity.TransactionTransfer && toAccountID != nil {
		balanceQuery := fmt.Sprintf(`
			UPDATE %s.financial_accounts SET balance = balance - $1, updated_at = now()
			WHERE id = $2 AND is_deleted = false
		`, tenantSchema)
		if _, err := dbTx.Exec(ctx, balanceQuery, amount, *toAccountID); err != nil {
			return fmt.Errorf("gagal balik saldo tujuan: %w", err)
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
		SELECT t.id, t.sync_id, t.account_id, t.category_id, t.transaction_type,
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
			&t.ID, &t.SyncID, &t.AccountID, &t.CategoryID, &t.TransactionType,
			&t.AmountIDR, &t.TransactionDate, &t.Notes, &t.ToAccountID,
			&t.IsDeleted, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan export row: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}
