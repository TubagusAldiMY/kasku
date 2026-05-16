package handler

import (
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/dto"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/response"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditLogHandler menangani GET /v1/admin/audit-log.
type AuditLogHandler struct {
	list usecase.ListAuditLogUseCase
}

// NewAuditLogHandler membuat instance.
func NewAuditLogHandler(list usecase.ListAuditLogUseCase) *AuditLogHandler {
	return &AuditLogHandler{list: list}
}

// List menangani GET /v1/admin/audit-log.
func (h *AuditLogHandler) List(c *gin.Context) {
	page, pageSize := paginationFromQuery(c)
	filter := repository.AuditLogFilter{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
	if v := c.Query("admin_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.AdminID = &id
		}
	}
	if v := c.Query("action"); v != "" {
		a := entity.AuditAction(v)
		filter.Action = &a
	}
	if v := c.Query("target_user_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.TargetUserID = &id
		}
	}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.To = &t
		}
	}

	out, err := h.list.Execute(c.Request.Context(), filter)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	items := make([]dto.AuditLogItem, 0, len(out.Entries))
	for _, e := range out.Entries {
		item := dto.AuditLogItem{
			ID:           e.ID.String(),
			AdminID:      e.AdminID.String(),
			Action:       string(e.Action),
			Metadata:     e.Metadata,
			Success:      e.Success,
			CreatedAt:    e.CreatedAt,
			TargetEntity: e.TargetEntity,
		}
		if e.TargetUserID != nil {
			s := e.TargetUserID.String()
			item.TargetUserID = &s
		}
		items = append(items, item)
	}
	response.OKMeta(c, items, dto.PaginationMeta{Page: page, PageSize: pageSize, Total: out.Total})
}
