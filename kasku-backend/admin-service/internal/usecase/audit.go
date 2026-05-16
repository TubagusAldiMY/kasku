package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AuditLogger membungkus AuditLogRepository dengan helper yang membuat entry konsisten.
// Setiap mutation use case wajib memanggil Log() supaya tidak ada aksi tanpa jejak.
type AuditLogger struct {
	repo repository.AuditLogRepository
	log  zerolog.Logger
}

// NewAuditLogger membuat helper audit dengan repository + logger.
func NewAuditLogger(repo repository.AuditLogRepository, log zerolog.Logger) *AuditLogger {
	return &AuditLogger{repo: repo, log: log}
}

// AuditInput adalah input pencatatan satu aksi admin.
type AuditInput struct {
	AdminID      uuid.UUID
	Action       entity.AuditAction
	TargetUserID *uuid.UUID
	TargetEntity *string
	Metadata     map[string]any
	Success      bool
}

// Log mencatat entry ke admin_audit_log. Kegagalan menulis log dianggap
// non-fatal untuk operasi caller — di-log dengan level error, tapi
// caller tidak boleh dibatalkan karenanya (memilih ketersediaan vs auditability).
func (a *AuditLogger) Log(ctx context.Context, in AuditInput) {
	metadata, err := json.Marshal(in.Metadata)
	if err != nil {
		a.log.Error().Err(err).Msg("gagal marshal metadata audit log")
		metadata = []byte("{}")
	}

	entry := &entity.AuditLogEntry{
		ID:           uuid.New(),
		AdminID:      in.AdminID,
		Action:       in.Action,
		TargetUserID: in.TargetUserID,
		TargetEntity: in.TargetEntity,
		Metadata:     metadata,
		Success:      in.Success,
		CreatedAt:    time.Now().UTC(),
	}

	if err := a.repo.Create(ctx, entry); err != nil {
		a.log.Error().
			Err(err).
			Str("admin_id", in.AdminID.String()).
			Str("action", string(in.Action)).
			Bool("success", in.Success).
			Msg("gagal tulis audit log")
	}
}
