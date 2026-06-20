package handler

import (
	"net/http"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type DebtHandler struct {
	createUC    *usecase.CreateDebtUseCase
	listUC      *usecase.ListDebtsUseCase
	getUC       *usecase.GetDebtUseCase
	updateUC    *usecase.UpdateDebtUseCase
	deleteUC    *usecase.DeleteDebtUseCase
	recordPayUC *usecase.RecordDebtPaymentUseCase
	listPayUC   *usecase.ListDebtPaymentsUseCase
	log         zerolog.Logger
}

func NewDebtHandler(
	createUC *usecase.CreateDebtUseCase,
	listUC *usecase.ListDebtsUseCase,
	getUC *usecase.GetDebtUseCase,
	updateUC *usecase.UpdateDebtUseCase,
	deleteUC *usecase.DeleteDebtUseCase,
	recordPayUC *usecase.RecordDebtPaymentUseCase,
	listPayUC *usecase.ListDebtPaymentsUseCase,
	log zerolog.Logger,
) *DebtHandler {
	return &DebtHandler{
		createUC:    createUC,
		listUC:      listUC,
		getUC:       getUC,
		updateUC:    updateUC,
		deleteUC:    deleteUC,
		recordPayUC: recordPayUC,
		listPayUC:   listPayUC,
		log:         log,
	}
}

func (h *DebtHandler) extractCtx(c *gin.Context) (userID, tenantSchema string, ok bool) {
	userID = c.GetHeader(headerUserID)
	tenantSchema = c.GetHeader(headerTenantSchema)
	if userID == "" || tenantSchema == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Header autentikasi tidak ditemukan."},
		})
		return "", "", false
	}
	return userID, tenantSchema, true
}

func (h *DebtHandler) handleErr(c *gin.Context, err error) {
	if de, ok := domainerrors.IsDomainError(err); ok {
		switch de.Code {
		case domainerrors.ErrDebtNotFound.Code:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		case domainerrors.ErrDebtAlreadySettled.Code, domainerrors.ErrPaymentExceedsDebt.Code, domainerrors.ErrInvalidInput.Code:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		}
		return
	}
	correlationID, _ := c.Get("correlation_id")
	h.log.Error().Err(err).Interface("correlation_id", correlationID).Msg("debt handler error")
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Terjadi kesalahan internal."},
	})
}

func (h *DebtHandler) ListDebts(c *gin.Context) {
	userID, tenantSchema, ok := h.extractCtx(c)
	if !ok {
		return
	}

	var direction *entity.DebtDirection
	if d := c.Query("direction"); d == "RECEIVABLE" || d == "PAYABLE" {
		dir := entity.DebtDirection(d)
		direction = &dir
	}

	debts, err := h.listUC.Execute(c.Request.Context(), usecase.ListDebtsInput{
		TenantSchema: tenantSchema,
		UserID:       userID,
		Direction:    direction,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": debts})
}

func (h *DebtHandler) CreateDebt(c *gin.Context) {
	userID, tenantSchema, ok := h.extractCtx(c)
	if !ok {
		return
	}

	var req struct {
		Direction  string  `json:"direction" binding:"required"`
		PersonName string  `json:"person_name" binding:"required"`
		Amount     int64   `json:"amount" binding:"required"`
		DueDate    *string `json:"due_date"`
		Notes      string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": "Format due_date tidak valid (YYYY-MM-DD)."}})
			return
		}
		dueDate = &t
	}

	debt, err := h.createUC.Execute(c.Request.Context(), usecase.CreateDebtInput{
		TenantSchema: tenantSchema,
		UserID:       userID,
		Direction:    entity.DebtDirection(req.Direction),
		PersonName:   req.PersonName,
		TotalAmount:  req.Amount,
		DueDate:      dueDate,
		Notes:        req.Notes,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": debt})
}

func (h *DebtHandler) GetDebt(c *gin.Context) {
	userID, tenantSchema, ok := h.extractCtx(c)
	if !ok {
		return
	}
	debt, err := h.getUC.Execute(c.Request.Context(), tenantSchema, c.Param("id"), userID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": debt})
}

func (h *DebtHandler) UpdateDebt(c *gin.Context) {
	userID, tenantSchema, ok := h.extractCtx(c)
	if !ok {
		return
	}

	var req struct {
		PersonName string  `json:"person_name" binding:"required"`
		DueDate    *string `json:"due_date"`
		Notes      string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": "Format due_date tidak valid (YYYY-MM-DD)."}})
			return
		}
		dueDate = &t
	}

	debt, err := h.updateUC.Execute(c.Request.Context(), usecase.UpdateDebtInput{
		TenantSchema: tenantSchema,
		ID:           c.Param("id"),
		UserID:       userID,
		PersonName:   req.PersonName,
		DueDate:      dueDate,
		Notes:        req.Notes,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": debt})
}

func (h *DebtHandler) DeleteDebt(c *gin.Context) {
	userID, tenantSchema, ok := h.extractCtx(c)
	if !ok {
		return
	}
	if err := h.deleteUC.Execute(c.Request.Context(), tenantSchema, c.Param("id"), userID); err != nil {
		h.handleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Catatan hutang berhasil dihapus."})
}

func (h *DebtHandler) RecordPayment(c *gin.Context) {
	userID, tenantSchema, ok := h.extractCtx(c)
	if !ok {
		return
	}

	var req struct {
		Amount      int64  `json:"amount" binding:"required"`
		PaymentDate string `json:"payment_date"`
		Notes       string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	paymentDate := time.Now().UTC()
	if req.PaymentDate != "" {
		t, err := time.Parse("2006-01-02", req.PaymentDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": "Format payment_date tidak valid (YYYY-MM-DD)."}})
			return
		}
		paymentDate = t
	}

	payment, err := h.recordPayUC.Execute(c.Request.Context(), usecase.RecordDebtPaymentInput{
		TenantSchema: tenantSchema,
		DebtID:       c.Param("id"),
		UserID:       userID,
		Amount:       req.Amount,
		PaymentDate:  paymentDate,
		Notes:        req.Notes,
	})
	if err != nil {
		h.handleErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": payment})
}

func (h *DebtHandler) ListPayments(c *gin.Context) {
	userID, tenantSchema, ok := h.extractCtx(c)
	if !ok {
		return
	}
	payments, err := h.listPayUC.Execute(c.Request.Context(), tenantSchema, c.Param("id"), userID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": payments})
}
