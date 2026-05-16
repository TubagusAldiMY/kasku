package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
)

// ListAuditLogOutput membawa data + total count.
type ListAuditLogOutput struct {
	Entries []entity.AuditLogEntry
	Total   int64
}

// ListAuditLogUseCase mengambil daftar audit log dari kasku_admin.
type ListAuditLogUseCase interface {
	Execute(ctx context.Context, filter repository.AuditLogFilter) (*ListAuditLogOutput, error)
}

type listAuditLogUseCase struct {
	repo repository.AuditLogRepository
}

// NewListAuditLogUseCase membuat instance.
func NewListAuditLogUseCase(repo repository.AuditLogRepository) ListAuditLogUseCase {
	return &listAuditLogUseCase{repo: repo}
}

func (uc *listAuditLogUseCase) Execute(ctx context.Context, filter repository.AuditLogFilter) (*ListAuditLogOutput, error) {
	entries, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &ListAuditLogOutput{Entries: entries, Total: total}, nil
}
