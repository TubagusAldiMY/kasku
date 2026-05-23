package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	headerUserID              = "X-User-ID"
	headerTenantSchema        = "X-Tenant-Schema"
	headerTierMaxTransactions = "X-Tier-Max-Transactions"
	headerTierHistoryMonths   = "X-Tier-History-Months"
	headerTierExportCSV       = "X-Tier-Export-CSV"
	headerTierMaxBudgets      = "X-Tier-Max-Budgets"
)

// HealthChecker adalah port untuk dependency health check (menghindari import persistence dari handler).
type HealthChecker interface {
	PingPostgres(ctx context.Context) error
}

type TransactionHandler struct {
	createTxUC     *usecase.CreateTransactionUseCase
	listTxUC       *usecase.ListTransactionsUseCase
	getTxUC        *usecase.GetTransactionUseCase
	updateTxUC     *usecase.UpdateTransactionUseCase
	deleteTxUC     *usecase.DeleteTransactionUseCase
	exportCSVUC    *usecase.ExportCSVUseCase
	listCatUC      *usecase.ListCategoriesUseCase
	createCatUC    *usecase.CreateCategoryUseCase
	updateCatUC    *usecase.UpdateCategoryUseCase
	deleteCatUC    *usecase.DeleteCategoryUseCase
	createBudgetUC *usecase.CreateBudgetUseCase
	listBudgetsUC  *usecase.ListBudgetsUseCase
	getBudgetUC    *usecase.GetBudgetUseCase
	updateBudgetUC *usecase.UpdateBudgetUseCase
	deleteBudgetUC *usecase.DeleteBudgetUseCase
	health         HealthChecker
	serviceVersion string
	log            zerolog.Logger
}

func NewTransactionHandler(
	createTxUC *usecase.CreateTransactionUseCase,
	listTxUC *usecase.ListTransactionsUseCase,
	getTxUC *usecase.GetTransactionUseCase,
	updateTxUC *usecase.UpdateTransactionUseCase,
	deleteTxUC *usecase.DeleteTransactionUseCase,
	exportCSVUC *usecase.ExportCSVUseCase,
	listCatUC *usecase.ListCategoriesUseCase,
	createCatUC *usecase.CreateCategoryUseCase,
	updateCatUC *usecase.UpdateCategoryUseCase,
	deleteCatUC *usecase.DeleteCategoryUseCase,
	createBudgetUC *usecase.CreateBudgetUseCase,
	listBudgetsUC *usecase.ListBudgetsUseCase,
	getBudgetUC *usecase.GetBudgetUseCase,
	updateBudgetUC *usecase.UpdateBudgetUseCase,
	deleteBudgetUC *usecase.DeleteBudgetUseCase,
	health HealthChecker,
	serviceVersion string,
	log zerolog.Logger,
) *TransactionHandler {
	return &TransactionHandler{
		createTxUC:     createTxUC,
		listTxUC:       listTxUC,
		getTxUC:        getTxUC,
		updateTxUC:     updateTxUC,
		deleteTxUC:     deleteTxUC,
		exportCSVUC:    exportCSVUC,
		listCatUC:      listCatUC,
		createCatUC:    createCatUC,
		updateCatUC:    updateCatUC,
		deleteCatUC:    deleteCatUC,
		createBudgetUC: createBudgetUC,
		listBudgetsUC:  listBudgetsUC,
		getBudgetUC:    getBudgetUC,
		updateBudgetUC: updateBudgetUC,
		deleteBudgetUC: deleteBudgetUC,
		health:         health,
		serviceVersion: serviceVersion,
		log:            log,
	}
}

// ─── Health ───────────────────────────────────────────────────────────────────

func (h *TransactionHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	checks := gin.H{}
	httpStatus := http.StatusOK
	status := "healthy"
	if err := h.health.PingPostgres(ctx); err != nil {
		checks["postgres"] = "unhealthy"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["postgres"] = "healthy"
	}
	c.JSON(httpStatus, gin.H{"status": status, "version": h.serviceVersion, "checks": checks})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *TransactionHandler) extractRequestContext(c *gin.Context) (userID, tenantSchema string, ok bool) {
	userID = c.GetHeader(headerUserID)
	tenantSchema = c.GetHeader(headerTenantSchema)
	if userID == "" || tenantSchema == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Header autentikasi tidak ditemukan.",
			},
		})
		return "", "", false
	}
	return userID, tenantSchema, true
}

func (h *TransactionHandler) handleDomainError(c *gin.Context, err error) {
	correlationID, _ := c.Get("correlation_id")
	if de, ok := domainerrors.IsDomainError(err); ok {
		switch de.Code {
		case "TRANSACTION_NOT_FOUND", "CATEGORY_NOT_FOUND", "BUDGET_NOT_FOUND", "ACCOUNT_NOT_FOUND":
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		case "TRANSACTION_LIMIT_REACHED", "EXPORT_NOT_ALLOWED", "BUDGET_LIMIT_REACHED":
			c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		case "CATEGORY_HAS_TRANSACTIONS", "DEFAULT_CATEGORY_CANNOT_BE_DELETED":
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		}
		return
	}
	h.log.Error().Err(err).Interface("correlation_id", correlationID).Msg("unhandled internal error")
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Terjadi kesalahan internal."},
	})
}

func parseIntHeader(c *gin.Context, key string, defaultValue int) int {
	v := c.GetHeader(key)
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return n
}

// ─── Transactions ─────────────────────────────────────────────────────────────

func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	var from, to time.Time
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = t
		}
	}

	result, err := h.listTxUC.Execute(c.Request.Context(), usecase.ListTransactionsInput{
		TenantSchema:  tenantSchema,
		UserID:        userID,
		From:          from,
		To:            to,
		HistoryMonths: parseIntHeader(c, headerTierHistoryMonths, -1),
	})
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result.Transactions, "summary": result.Summary})
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	var req struct {
		SyncID          string                 `json:"sync_id"`
		AccountID       string                 `json:"account_id" binding:"required"`
		CategoryID      string                 `json:"category_id"`
		BudgetID        string                 `json:"budget_id"`
		TransactionType entity.TransactionType `json:"transaction_type" binding:"required"`
		AmountIDR       int64                  `json:"amount_idr" binding:"required"`
		TransactionDate string                 `json:"transaction_date"`
		Notes           string                 `json:"notes"`
		ToAccountID     string                 `json:"to_account_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	txDate := time.Now().UTC()
	if req.TransactionDate != "" {
		if t, err := time.Parse("2006-01-02", req.TransactionDate); err == nil {
			txDate = t
		}
	}

	tx, err := h.createTxUC.Execute(c.Request.Context(), usecase.CreateTransactionInput{
		TenantSchema:    tenantSchema,
		UserID:          userID,
		SyncID:          req.SyncID,
		AccountID:       req.AccountID,
		CategoryID:      req.CategoryID,
		BudgetID:        req.BudgetID,
		TransactionType: req.TransactionType,
		AmountIDR:       req.AmountIDR,
		TransactionDate: txDate,
		Notes:           req.Notes,
		ToAccountID:     req.ToAccountID,
		MaxTransactions: parseIntHeader(c, headerTierMaxTransactions, -1),
	})
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": tx})
}

func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	id := c.Param("id")

	var req struct {
		AccountID       string                 `json:"account_id" binding:"required"`
		CategoryID      string                 `json:"category_id"`
		BudgetID        string                 `json:"budget_id"`
		TransactionType entity.TransactionType `json:"transaction_type" binding:"required"`
		AmountIDR       int64                  `json:"amount_idr" binding:"required"`
		TransactionDate string                 `json:"transaction_date"`
		Notes           string                 `json:"notes"`
		ToAccountID     string                 `json:"to_account_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	txDate := time.Now().UTC()
	if req.TransactionDate != "" {
		if t, err := time.Parse("2006-01-02", req.TransactionDate); err == nil {
			txDate = t
		}
	}

	tx, err := h.updateTxUC.Execute(c.Request.Context(), usecase.UpdateTransactionInput{
		TenantSchema:    tenantSchema,
		UserID:          userID,
		ID:              id,
		AccountID:       req.AccountID,
		CategoryID:      req.CategoryID,
		BudgetID:        req.BudgetID,
		TransactionType: req.TransactionType,
		AmountIDR:       req.AmountIDR,
		TransactionDate: txDate,
		Notes:           req.Notes,
		ToAccountID:     req.ToAccountID,
	})
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tx})
}

func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	tx, err := h.getTxUC.Execute(c.Request.Context(), tenantSchema, id, userID)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tx})
}

func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.deleteTxUC.Execute(c.Request.Context(), tenantSchema, id, userID); err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Transaksi berhasil dihapus."})
}

func (h *TransactionHandler) ExportCSV(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	exportAllowed := c.GetHeader(headerTierExportCSV) == "true"

	data, err := h.exportCSVUC.Execute(c.Request.Context(), tenantSchema, userID, exportAllowed)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=transactions.csv")
	c.Data(http.StatusOK, "text/csv", data)
}

// ─── Categories ───────────────────────────────────────────────────────────────

func (h *TransactionHandler) ListCategories(c *gin.Context) {
	_, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	cats, err := h.listCatUC.Execute(c.Request.Context(), tenantSchema)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cats})
}

func (h *TransactionHandler) CreateCategory(c *gin.Context) {
	_, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	var req struct {
		Name         string              `json:"name" binding:"required"`
		Icon         string              `json:"icon"`
		Color        string              `json:"color"`
		CategoryType entity.CategoryType `json:"category_type" binding:"required,oneof=INCOME EXPENSE BOTH"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}
	cat, err := h.createCatUC.Execute(c.Request.Context(), usecase.CreateCategoryInput{
		TenantSchema: tenantSchema,
		Name:         req.Name,
		Icon:         req.Icon,
		Color:        req.Color,
		CategoryType: req.CategoryType,
	})
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": cat})
}

func (h *TransactionHandler) UpdateCategory(c *gin.Context) {
	_, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var req struct {
		Name         string              `json:"name" binding:"required"`
		Icon         string              `json:"icon"`
		Color        string              `json:"color"`
		CategoryType entity.CategoryType `json:"category_type" binding:"required,oneof=INCOME EXPENSE BOTH"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}
	if err := h.updateCatUC.Execute(c.Request.Context(), tenantSchema, id, req.Name, req.Icon, req.Color, req.CategoryType); err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Kategori berhasil diperbarui."})
}

func (h *TransactionHandler) DeleteCategory(c *gin.Context) {
	_, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.deleteCatUC.Execute(c.Request.Context(), tenantSchema, id); err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Kategori berhasil dihapus."})
}

// ─── Budgets ──────────────────────────────────────────────────────────────────

type budgetResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	LimitIDR        int64   `json:"limit_idr"`
	CategoryID      *string `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	PeriodType      string  `json:"period_type"`
	AlertThreshold  int     `json:"alert_threshold"`
	SpentIDR        int64   `json:"spent_idr"`
	RemainingIDR    int64   `json:"remaining_idr"`
	ProgressPercent float64 `json:"progress_percent"`
	IsOverBudget    bool    `json:"is_over_budget"`
	UpdatedAt       string  `json:"updated_at"`
	// Daily allowance fields — present only when daily_limit_enabled = true.
	DailyLimitEnabled      bool   `json:"daily_limit_enabled"`
	DailyBaseIDR           *int64 `json:"daily_base_idr,omitempty"`
	CarryoverIDR           *int64 `json:"carryover_idr,omitempty"`
	DailyAllowanceTodayIDR *int64 `json:"daily_allowance_today_idr,omitempty"`
	SpentTodayIDR          *int64 `json:"spent_today_idr,omitempty"`
	DailyRemainingIDR      *int64 `json:"daily_remaining_idr,omitempty"`
}

func mapBudgetResponse(b *entity.BudgetWithProgress) budgetResponse {
	var catID *string
	if b.CategoryID != nil {
		s := b.CategoryID.String()
		catID = &s
	}
	remaining := b.LimitIDR - b.SpentIDR
	var progress float64
	if b.LimitIDR > 0 {
		progress = float64(b.SpentIDR) / float64(b.LimitIDR) * 100
	}
	resp := budgetResponse{
		ID:                b.ID.String(),
		Name:              b.Name,
		LimitIDR:          b.LimitIDR,
		CategoryID:        catID,
		CategoryName:      b.CategoryName,
		PeriodType:        string(b.PeriodType),
		AlertThreshold:    b.AlertThreshold,
		SpentIDR:          b.SpentIDR,
		RemainingIDR:      remaining,
		ProgressPercent:   progress,
		IsOverBudget:      b.SpentIDR > b.LimitIDR,
		UpdatedAt:         b.UpdatedAt.Format(time.RFC3339),
		DailyLimitEnabled: b.DailyLimitEnabled,
	}
	if b.DailyLimitEnabled {
		resp.DailyBaseIDR = &b.DailyBaseIDR
		resp.CarryoverIDR = &b.CarryoverIDR
		resp.DailyAllowanceTodayIDR = &b.DailyAllowanceTodayIDR
		resp.SpentTodayIDR = &b.SpentTodayIDR
		resp.DailyRemainingIDR = &b.DailyRemainingIDR
	}
	return resp
}

func (h *TransactionHandler) ListBudgets(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	budgets, err := h.listBudgetsUC.Execute(c.Request.Context(), tenantSchema, userID)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	resp := make([]budgetResponse, 0, len(budgets))
	for i := range budgets {
		resp = append(resp, mapBudgetResponse(&budgets[i]))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TransactionHandler) CreateBudget(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	var req struct {
		SyncID            string `json:"sync_id"`
		Name              string `json:"name"           binding:"required,max=100"`
		LimitIDR          int64  `json:"limit_idr"      binding:"required,min=1"`
		CategoryID        string `json:"category_id"`
		PeriodType        string `json:"period_type"`
		StartDate         string `json:"start_date"`
		EndDate           string `json:"end_date"`
		AlertThreshold    int    `json:"alert_threshold"`
		DailyLimitEnabled bool   `json:"daily_limit_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": "X-User-ID tidak valid."}})
		return
	}

	var catID *uuid.UUID
	if req.CategoryID != "" {
		id, err := parseUUID(req.CategoryID)
		if err == nil {
			catID = &id
		}
	}

	var startDate time.Time
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = t
		}
	}
	var endDate *time.Time
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endDate = &t
		}
	}

	budget, err := h.createBudgetUC.Execute(c.Request.Context(), usecase.CreateBudgetInput{
		TenantSchema:      tenantSchema,
		UserID:            userUUID,
		SyncID:            req.SyncID,
		Name:              req.Name,
		LimitIDR:          req.LimitIDR,
		CategoryID:        catID,
		PeriodType:        entity.BudgetPeriodType(req.PeriodType),
		StartDate:         startDate,
		EndDate:           endDate,
		AlertThreshold:    req.AlertThreshold,
		DailyLimitEnabled: req.DailyLimitEnabled,
		MaxBudgets:        parseIntHeader(c, headerTierMaxBudgets, -1),
	})
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": gin.H{"id": budget.ID.String()}})
}

func (h *TransactionHandler) GetBudget(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	b, err := h.getBudgetUC.Execute(c.Request.Context(), tenantSchema, id, userID)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": mapBudgetResponse(b)})
}

func (h *TransactionHandler) UpdateBudget(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var req struct {
		Name              string `json:"name"      binding:"required,max=100"`
		LimitIDR          int64  `json:"limit_idr" binding:"required,min=1"`
		CategoryID        string `json:"category_id"`
		AlertThreshold    int    `json:"alert_threshold"`
		DailyLimitEnabled bool   `json:"daily_limit_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": "X-User-ID tidak valid."}})
		return
	}
	budgetUUID, err := parseUUID(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_INPUT", "message": "ID anggaran tidak valid."}})
		return
	}

	var catID *uuid.UUID
	if req.CategoryID != "" {
		parsed, err := parseUUID(req.CategoryID)
		if err == nil {
			catID = &parsed
		}
	}

	if err := h.updateBudgetUC.Execute(c.Request.Context(), usecase.UpdateBudgetInput{
		TenantSchema:      tenantSchema,
		UserID:            userUUID,
		ID:                budgetUUID,
		Name:              req.Name,
		LimitIDR:          req.LimitIDR,
		CategoryID:        catID,
		AlertThreshold:    req.AlertThreshold,
		DailyLimitEnabled: req.DailyLimitEnabled,
	}); err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Anggaran berhasil diperbarui."})
}

func (h *TransactionHandler) DeleteBudget(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.deleteBudgetUC.Execute(c.Request.Context(), tenantSchema, id, userID); err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Anggaran berhasil dihapus."})
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
