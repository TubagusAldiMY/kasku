package persistence

import (
	"context"
	"fmt"
	"strings"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresAuditLogRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAuditLogRepository membuat repository admin_audit_log (kasku_admin DB).
func NewPostgresAuditLogRepository(pool *pgxpool.Pool) repository.AuditLogRepository {
	return &postgresAuditLogRepository{pool: pool}
}

func (r *postgresAuditLogRepository) Create(ctx context.Context, e *entity.AuditLogEntry) error {
	const q = `
		INSERT INTO public.admin_audit_log
			(id, admin_id, action, target_user_id, target_entity, metadata, success)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, q,
		e.ID, e.AdminID, string(e.Action), e.TargetUserID, e.TargetEntity, e.Metadata, e.Success,
	)
	if err != nil {
		return fmt.Errorf("gagal insert audit log: %w", err)
	}
	return nil
}

func (r *postgresAuditLogRepository) List(ctx context.Context, f repository.AuditLogFilter) ([]entity.AuditLogEntry, int64, error) {
	var (
		conds []string
		args  []any
		i     = 1
	)

	if f.AdminID != nil {
		conds = append(conds, fmt.Sprintf("admin_id = $%d", i))
		args = append(args, *f.AdminID)
		i++
	}
	if f.Action != nil {
		conds = append(conds, fmt.Sprintf("action = $%d", i))
		args = append(args, string(*f.Action))
		i++
	}
	if f.TargetUserID != nil {
		conds = append(conds, fmt.Sprintf("target_user_id = $%d", i))
		args = append(args, *f.TargetUserID)
		i++
	}
	if f.From != nil {
		conds = append(conds, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, *f.From)
		i++
	}
	if f.To != nil {
		conds = append(conds, fmt.Sprintf("created_at < $%d", i))
		args = append(args, *f.To)
		i++
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	// Count total
	var total int64
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM public.admin_audit_log %s", where)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal count audit log: %w", err)
	}

	// Page query
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	listQ := fmt.Sprintf(`
		SELECT id, admin_id, action, target_user_id, target_entity, metadata, success, created_at
		FROM public.admin_audit_log
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, i, i+1)

	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal query audit log: %w", err)
	}
	defer rows.Close()

	out := make([]entity.AuditLogEntry, 0, limit)
	for rows.Next() {
		var e entity.AuditLogEntry
		var actionStr string
		if err := rows.Scan(&e.ID, &e.AdminID, &actionStr, &e.TargetUserID, &e.TargetEntity, &e.Metadata, &e.Success, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("gagal scan audit log: %w", err)
		}
		e.Action = entity.AuditAction(actionStr)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterasi audit log: %w", err)
	}
	return out, total, nil
}
