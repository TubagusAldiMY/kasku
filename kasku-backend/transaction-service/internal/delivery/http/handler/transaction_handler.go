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
	"github.com/rs/zerolog"
)

const (
	headerUserID              = "X-User-ID"
	headerTenantSchema        = "X-Tenant-Schema"
	headerTierMaxTransactions = "X-Tier-Max-Transactions"
	headerTierHistoryMonths   = "X-Tier-History-Months"
	headerTierExportCSV       = "X-Tier-Export-CSV"
)

// HealthChecker adalah port untuk dependency health check (menghindari import persistence dari handler).
type HealthChecker interface {
	PingPostgres(ctx context.Context) error
}

type TransactionHandler struct {
	createTxUC     *usecase.CreateTransactionUseCase
	listTxUC       *usecase.ListTransactionsUseCase
	getTxUC        *usecase.GetTransactionUseCase
	deleteTxUC     *usecase.DeleteTransactionUseCase
	exportCSVUC    *usecase.ExportCSVUseCase
	listCatUC      *usecase.ListCategoriesUseCase
	createCatUC    *usecase.CreateCategoryUseCase
	updateCatUC    *usecase.UpdateCategoryUseCase
	deleteCatUC    *usecase.DeleteCategoryUseCase
	health         HealthChecker
	serviceVersion string
	log            zerolog.Logger
}

func NewTransactionHandler(
	createTxUC *usecase.CreateTransactionUseCase,
	listTxUC *usecase.ListTransactionsUseCase,
	getTxUC *usecase.GetTransactionUseCase,
	deleteTxUC *usecase.DeleteTransactionUseCase,
	exportCSVUC *usecase.ExportCSVUseCase,
	listCatUC *usecase.ListCategoriesUseCase,
	createCatUC *usecase.CreateCategoryUseCase,
	updateCatUC *usecase.UpdateCategoryUseCase,
	deleteCatUC *usecase.DeleteCategoryUseCase,
	health HealthChecker,
	serviceVersion string,
	log zerolog.Logger,
) *TransactionHandler {
	return &TransactionHandler{
		createTxUC:     createTxUC,
		listTxUC:       listTxUC,
		getTxUC:        getTxUC,
		deleteTxUC:     deleteTxUC,
		exportCSVUC:    exportCSVUC,
		listCatUC:      listCatUC,
		createCatUC:    createCatUC,
		updateCatUC:    updateCatUC,
		deleteCatUC:    deleteCatUC,
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
		case "TRANSACTION_NOT_FOUND", "CATEGORY_NOT_FOUND":
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		case "TRANSACTION_LIMIT_REACHED", "EXPORT_NOT_ALLOWED":
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
