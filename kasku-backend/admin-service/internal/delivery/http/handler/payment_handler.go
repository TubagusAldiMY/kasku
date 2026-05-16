package handler

import (
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/dto"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/response"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentHandler menangani list payment di admin dashboard.
type PaymentHandler struct {
	list usecase.ListPaymentsUseCase
}

// NewPaymentHandler membuat instance.
func NewPaymentHandler(list usecase.ListPaymentsUseCase) *PaymentHandler {
	return &PaymentHandler{list: list}
}

// List menangani GET /v1/admin/payments.
func (h *PaymentHandler) List(c *gin.Context) {
	page, pageSize := paginationFromQuery(c)
	filter := repository.PaymentListFilter{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
	if v := c.Query("user_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.UserID = &id
		}
	}
	if v := c.Query("status"); v != "" {
		filter.Status = &v
	}
	if v := c.Query("tier"); v != "" {
		filter.PlanName = &v
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

	items := make([]dto.PaymentListItem, 0, len(out.Payments))
	for _, p := range out.Payments {
		items = append(items, dto.PaymentListItem{
			ID:        p.ID.String(),
			UserID:    p.UserID.String(),
			OrderID:   p.OrderID,
			AmountIDR: p.AmountIDR,
			Status:    p.Status,
			PlanName:  p.PlanName,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	response.OKMeta(c, items, dto.PaginationMeta{Page: page, PageSize: pageSize, Total: out.Total})
}
