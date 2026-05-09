package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FinanceRepository mendefinisikan operasi ke kasku_finance.
type FinanceRepository interface {
	ProvisionTenant(ctx context.Context, userID string) error
}

type postgresFinanceRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresFinanceRepository(pool *pgxpool.Pool) FinanceRepository {
	return &postgresFinanceRepository{pool: pool}
}

// ProvisionTenant memanggil stored function provision_tenant di kasku_finance.
// Fungsi ini idempotent — aman dipanggil berulang kali untuk user yang sama.
func (r *postgresFinanceRepository) ProvisionTenant(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, "SELECT provision_tenant($1::uuid)", userID)
	if err != nil {
		return fmt.Errorf("gagal panggil provision_tenant untuk user %s: %w", userID, err)
	}
	return nil
}
