package handler

import (
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/dto"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/response"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscriptionHandler menangani override subscription user.
type SubscriptionHandler struct {
	override usecase.OverrideSubscriptionUseCase
}

// NewSubscriptionHandler membuat instance.
func NewSubscriptionHandler(override usecase.OverrideSubscriptionUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{override: override}
}

// Override menangani POST /v1/admin/users/:id/override-subscription.
func (h *SubscriptionHandler) Override(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, domainerrors.ErrValidation.Code, "ID user tidak valid.")
		return
	}
	var req dto.OverrideSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, domainerrors.ErrValidation.Code, domainerrors.ErrValidation.Message)
		return
	}
	adminID, err := adminIDFromContext(c)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	if err := h.override.Execute(c.Request.Context(), usecase.OverrideSubscriptionInput{
		AdminID:      adminID,
		TargetUserID: targetID,
		NewPlanName:  req.PlanName,
		Reason:       req.Reason,
	}); err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "subscription di-override"})
}
