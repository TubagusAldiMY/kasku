package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// tenantSchemaPattern memvalidasi nama schema tenant sebelum diinterpolasi ke SQL.
// Format: tenant_<uuid tanpa dash, underscore-separated> — sama dengan yang
// dihasilkan provision_tenant. Ini pertahanan utama terhadap SQL injection lewat
// nama schema (identifier tidak bisa diparameterkan).
var tenantSchemaPattern = regexp.MustCompile(`^tenant_[0-9a-f_]{32,36}$`)

func validTenantSchema(schema string) bool {
	return tenantSchemaPattern.MatchString(schema)
}

// tenantExportTables adalah allowlist tabel per-tenant yang ikut diekspor.
// Hanya data milik user yang bermakna untuk portabilitas — tabel turunan/internal
// (balance_history, unit_history, sync_log) sengaja tidak diikutkan.
var tenantExportTables = []string{
	"financial_accounts",
	"transactions",
	"categories",
	"budgets",
	"debts",
	"investment_assets",
}

// ExportRepository mengumpulkan seluruh data user untuk portabilitas (UU PDP / GDPR).
type ExportRepository interface {
	// DumpTenantData mengembalikan map tabel->array-row(JSON) untuk tabel tenant
	// yang benar-benar ada (toleran terhadap tenant lama yang belum punya tabel baru).
	DumpTenantData(ctx context.Context, tenantSchema string) (map[string]json.RawMessage, error)
	// DumpSubscription mengembalikan baris langganan billing user sebagai JSON.
	DumpSubscription(ctx context.Context, userID string) (json.RawMessage, error)
}

type postgresExportRepository struct {
	financePool *pgxpool.Pool
	billingPool *pgxpool.Pool
}

func NewPostgresExportRepository(financePool, billingPool *pgxpool.Pool) ExportRepository {
	return &postgresExportRepository{financePool: financePool, billingPool: billingPool}
}

func (r *postgresExportRepository) DumpTenantData(ctx context.Context, tenantSchema string) (map[string]json.RawMessage, error) {
	if !validTenantSchema(tenantSchema) {
		return nil, fmt.Errorf("tenant schema tidak valid: %q", tenantSchema)
	}

	// Cari tabel allowlist yang benar-benar ada di schema ini.
	rows, err := r.financePool.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = $1 AND table_name = ANY($2)
	`, tenantSchema, tenantExportTables)
	if err != nil {
		return nil, fmt.Errorf("gagal list tabel tenant %s: %w", tenantSchema, err)
	}
	var existing []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return nil, fmt.Errorf("gagal scan nama tabel: %w", err)
		}
		existing = append(existing, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal iterasi tabel tenant: %w", err)
	}

	out := make(map[string]json.RawMessage, len(existing))
	for _, table := range existing {
		// schema (regex-validated) + table (dari allowlist) → aman diquote via Sanitize.
		ident := pgx.Identifier{tenantSchema, table}.Sanitize()
		q := fmt.Sprintf(`SELECT COALESCE(json_agg(to_jsonb(t)), '[]'::json) FROM %s t`, ident)
		var raw []byte
		if err := r.financePool.QueryRow(ctx, q).Scan(&raw); err != nil {
			return nil, fmt.Errorf("gagal dump tabel %s: %w", table, err)
		}
		out[table] = json.RawMessage(raw)
	}
	return out, nil
}

func (r *postgresExportRepository) DumpSubscription(ctx context.Context, userID string) (json.RawMessage, error) {
	var raw []byte
	err := r.billingPool.QueryRow(ctx, `
		SELECT COALESCE(json_agg(to_jsonb(s)), '[]'::json)
		FROM public.subscriptions s
		WHERE s.user_id = $1::uuid
	`, userID).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("gagal dump subscription user %s: %w", userID, err)
	}
	return json.RawMessage(raw), nil
}
