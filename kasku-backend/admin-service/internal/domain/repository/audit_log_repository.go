package repository

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/google/uuid"
)

// AuditLogFilter adalah opsi filter untuk list audit log.
type AuditLogFilter struct {
	AdminID      *uuid.UUID
	Action       *entity.AuditAction
	TargetUserID *uuid.UUID
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

// AuditLogRepository adalah port untuk admin_audit_log di kasku_admin.
type AuditLogRepository interface {
	Create(ctx context.Context, entry *entity.AuditLogEntry) error
	List(ctx context.Context, filter AuditLogFilter) ([]entity.AuditLogEntry, int64, error)
}
