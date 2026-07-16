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

type postgresDebtRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresDebtRepository(pool *pgxpool.Pool) repository.DebtRepository {
	return &postgresDebtRepository{pool: pool}
}

func (r *postgresDebtRepository) Create(ctx context.Context, tenantSchema string, debt *entity.Debt) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		INSERT INTO %s.debts
			(id, user_id, direction, person_name, total_amount, remaining_amount, due_date, notes, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, tenantSchema)
	_, err := r.pool.Exec(ctx, query,
		debt.ID, debt.UserID, string(debt.Direction), debt.PersonName,
		debt.TotalAmount, debt.RemainingAmount, debt.DueDate, debt.Notes,
		string(debt.Status), debt.CreatedAt, debt.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert debt: %w", err)
	}
	return nil
}

func (r *postgresDebtRepository) List(ctx context.Context, tenantSchema, userID string, direction *entity.DebtDirection) ([]entity.Debt, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}

	var (
		query string
		args  []interface{}
	)
	if direction != nil {
		query = fmt.Sprintf(`
			SELECT id, user_id, direction, person_name, total_amount, remaining_amount, due_date, notes, status, created_at, updated_at
			FROM %s.debts
			WHERE user_id = $1 AND direction = $2
			ORDER BY status ASC, created_at DESC
		`, tenantSchema)
		args = []interface{}{userID, string(*direction)}
	} else {
		query = fmt.Sprintf(`
			SELECT id, user_id, direction, person_name, total_amount, remaining_amount, due_date, notes, status, created_at, updated_at
			FROM %s.debts
			WHERE user_id = $1
			ORDER BY status ASC, created_at DESC
		`, tenantSchema)
		args = []interface{}{userID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query debts: %w", err)
	}
	defer rows.Close()

	var debts []entity.Debt
	for rows.Next() {
		d := entity.Debt{}
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Direction, &d.PersonName,
			&d.TotalAmount, &d.RemainingAmount, &d.DueDate, &d.Notes,
			&d.Status, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan debt: %w", err)
		}
		debts = append(debts, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi debts: %w", err)
	}
	return debts, nil
}

func (r *postgresDebtRepository) GetByID(ctx context.Context, tenantSchema, id, userID string) (*entity.Debt, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id, user_id, direction, person_name, total_amount, remaining_amount, due_date, notes, status, created_at, updated_at
		FROM %s.debts
		WHERE id = $1 AND user_id = $2
	`, tenantSchema)

	d := &entity.Debt{}
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&d.ID, &d.UserID, &d.Direction, &d.PersonName,
		&d.TotalAmount, &d.RemainingAmount, &d.DueDate, &d.Notes,
		&d.Status, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrDebtNotFound
		}
		return nil, fmt.Errorf("gagal get debt: %w", err)
	}
	return d, nil
}

func (r *postgresDebtRepository) Update(ctx context.Context, tenantSchema string, debt *entity.Debt) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`
		UPDATE %s.debts
		SET person_name = $3, due_date = $4, notes = $5, updated_at = $6
		WHERE id = $1 AND user_id = $2
	`, tenantSchema)
	result, err := r.pool.Exec(ctx, query,
		debt.ID, debt.UserID, debt.PersonName, debt.DueDate, debt.Notes, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("gagal update debt: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrDebtNotFound
	}
	return nil
}

func (r *postgresDebtRepository) Delete(ctx context.Context, tenantSchema, id, userID string) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}
	query := fmt.Sprintf(`DELETE FROM %s.debts WHERE id = $1 AND user_id = $2`, tenantSchema)
	result, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("gagal hapus debt: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrDebtNotFound
	}
	return nil
}

// RecordPayment mencatat pembayaran + mengurangi remaining_amount secara atomik.
// Row hutang dikunci dengan SELECT ... FOR UPDATE sehingga pembayaran konkuren
// diserialisasi — mencegah overpay/remaining_amount negatif. Validasi status &
// jumlah dilakukan di dalam transaksi terhadap nilai yang sudah terkunci.
func (r *postgresDebtRepository) RecordPayment(ctx context.Context, tenantSchema string, debtID, userID string, payment *entity.DebtPayment) error {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gagal mulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lockQuery := fmt.Sprintf(`
		SELECT remaining_amount, status
		FROM %s.debts
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, tenantSchema)

	var (
		remaining int64
		status    string
	)
	if err := tx.QueryRow(ctx, lockQuery, debtID, userID).Scan(&remaining, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domainerrors.ErrDebtNotFound
		}
		return fmt.Errorf("gagal lock debt: %w", err)
	}

	if status == string(entity.DebtStatusSettled) {
		return domainerrors.ErrDebtAlreadySettled
	}
	if payment.Amount > remaining {
		return domainerrors.ErrPaymentExceedsDebt
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO %s.debt_payments (id, debt_id, amount, payment_date, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantSchema)
	if _, err := tx.Exec(ctx, insertQuery,
		payment.ID, payment.DebtID, payment.Amount,
		payment.PaymentDate, payment.Notes, payment.CreatedAt,
	); err != nil {
		return fmt.Errorf("gagal insert payment: %w", err)
	}

	// Self-guarding update: WHERE remaining_amount >= $2 sebagai lapis pertahanan
	// kedua (row sudah terkunci, jadi ini selalu benar; tetap dipertahankan agar
	// invariant eksplisit di SQL). RowsAffected==0 menandakan race yang lolos.
	deductQuery := fmt.Sprintf(`
		UPDATE %s.debts
		SET
			remaining_amount = remaining_amount - $2,
			status = CASE WHEN remaining_amount - $2 <= 0 THEN 'SETTLED' ELSE status END,
			updated_at = now()
		WHERE id = $1 AND user_id = $3 AND remaining_amount >= $2
	`, tenantSchema)
	result, err := tx.Exec(ctx, deductQuery, debtID, payment.Amount, userID)
	if err != nil {
		return fmt.Errorf("gagal deduct remaining: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domainerrors.ErrPaymentExceedsDebt
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("gagal commit pembayaran: %w", err)
	}
	return nil
}

func (r *postgresDebtRepository) ListPayments(ctx context.Context, tenantSchema, debtID string) ([]entity.DebtPayment, error) {
	if err := ValidateTenantSchema(tenantSchema); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id, debt_id, amount, payment_date, notes, created_at
		FROM %s.debt_payments
		WHERE debt_id = $1
		ORDER BY created_at DESC
	`, tenantSchema)

	rows, err := r.pool.Query(ctx, query, debtID)
	if err != nil {
		return nil, fmt.Errorf("gagal query payments: %w", err)
	}
	defer rows.Close()

	var payments []entity.DebtPayment
	for rows.Next() {
		p := entity.DebtPayment{}
		if err := rows.Scan(&p.ID, &p.DebtID, &p.Amount, &p.PaymentDate, &p.Notes, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("gagal scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi payments: %w", err)
	}
	return payments, nil
}
