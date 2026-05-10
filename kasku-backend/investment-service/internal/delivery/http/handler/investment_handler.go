package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/investment-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	headerUserID             = "X-User-ID"
	headerTenantSchema       = "X-Tenant-Schema"
	headerTierMaxInvestments = "X-Tier-Max-Investments"
	headerTierHistoryMonths  = "X-Tier-History-Months"

	tierUnlimited = -1
)

// HealthChecker mendefinisikan kontrak untuk health check dependency.
type HealthChecker interface {
	PingPostgres(ctx context.Context) error
}

type InvestmentHandler struct {
	createUC   *usecase.CreateAssetUseCase
	listUC     *usecase.ListAssetsUseCase
	getUC      *usecase.GetAssetUseCase
	updateUC   *usecase.UpdateAssetUseCase
	deleteUC   *usecase.DeleteAssetUseCase
	recordUC   *usecase.RecordUnitChangeUseCase
	historyUC  *usecase.GetUnitHistoryUseCase
	health     HealthChecker
	svcVersion string
	log        zerolog.Logger
}

func NewInvestmentHandler(
	createUC *usecase.CreateAssetUseCase,
	listUC *usecase.ListAssetsUseCase,
	getUC *usecase.GetAssetUseCase,
	updateUC *usecase.UpdateAssetUseCase,
	deleteUC *usecase.DeleteAssetUseCase,
	recordUC *usecase.RecordUnitChangeUseCase,
	historyUC *usecase.GetUnitHistoryUseCase,
	health HealthChecker,
	svcVersion string,
	log zerolog.Logger,
) *InvestmentHandler {
	return &InvestmentHandler{
		createUC:   createUC,
		listUC:     listUC,
		getUC:      getUC,
		updateUC:   updateUC,
		deleteUC:   deleteUC,
		recordUC:   recordUC,
		historyUC:  historyUC,
		health:     health,
		svcVersion: svcVersion,
		log:        log,
	}
}

func (h *InvestmentHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	status := "healthy"
	httpStatus := http.StatusOK
	checks := gin.H{}

	if err := h.health.PingPostgres(ctx); err != nil {
		checks["postgres"] = "unhealthy"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["postgres"] = "healthy"
	}

	c.JSON(httpStatus, gin.H{
		"status":  status,
		"version": h.svcVersion,
		"checks":  checks,
	})
}

func (h *InvestmentHandler) extractRequestContext(c *gin.Context) (tenantSchema string, ok bool) {
	tenantSchema = c.GetHeader(headerTenantSchema)
	userID := c.GetHeader(headerUserID)
	if userID == "" || tenantSchema == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Header autentikasi tidak ditemukan."},
		})
		return "", false
	}
	return tenantSchema, true
}

func parseTierIntHeader(c *gin.Context, headerName string, defaultValue int) int {
	raw := c.GetHeader(headerName)
	if raw == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return val
}

func (h *InvestmentHandler) handleDomainError(c *gin.Context, err error) {
	if de, ok := domainerrors.IsDomainError(err); ok {
		switch de.Code {
		case domainerrors.ErrAssetNotFound.Code:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		case domainerrors.ErrAssetLimitReached.Code:
			c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		}
		return
	}
	h.log.Error().Err(err).Msg("unexpected internal error")
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Terjadi kesalahan internal."},
	})
}

func (h *InvestmentHandler) ListAssets(c *gin.Context) {
	tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	assets, err := h.listUC.Execute(c.Request.Context(), tenantSchema)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	if assets == nil {
		assets = []entity.InvestmentAsset{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": assets})
}

func (h *InvestmentHandler) CreateAsset(c *gin.Context) {
	tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	var req struct {
		Name        string           `json:"name" binding:"required"`
		AssetType   entity.AssetType `json:"asset_type" binding:"required"`
		Symbol      string           `json:"symbol" binding:"required"`
		Quantity    float64          `json:"quantity"`
		AvgBuyPrice float64          `json:"avg_buy_price"`
		Currency    string           `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_INPUT", "message": err.Error()},
		})
		return
	}

	maxInvestments := parseTierIntHeader(c, headerTierMaxInvestments, tierUnlimited)

	asset, err := h.createUC.Execute(c.Request.Context(), usecase.CreateAssetInput{
		TenantSchema:   tenantSchema,
		Name:           req.Name,
		AssetType:      req.AssetType,
		Symbol:         req.Symbol,
		Quantity:       req.Quantity,
		AvgBuyPrice:    req.AvgBuyPrice,
		Currency:       req.Currency,
		MaxInvestments: maxInvestments,
	})
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": asset})
}

func (h *InvestmentHandler) GetAsset(c *gin.Context) {
	tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	asset, err := h.getUC.Execute(c.Request.Context(), tenantSchema, id)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": asset})
}

func (h *InvestmentHandler) UpdateAsset(c *gin.Context) {
	tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var req struct {
		Name      string           `json:"name" binding:"required"`
		AssetType entity.AssetType `json:"asset_type" binding:"required"`
		Symbol    string           `json:"symbol" binding:"required"`
		Currency  string           `json:"currency"`
		SortOrder int              `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_INPUT", "message": err.Error()},
		})
		return
	}

	asset, err := h.updateUC.Execute(c.Request.Context(), usecase.UpdateAssetInput{
		TenantSchema: tenantSchema,
		ID:           id,
		Name:         req.Name,
		AssetType:    req.AssetType,
		Symbol:       req.Symbol,
		Currency:     req.Currency,
		SortOrder:    req.SortOrder,
	})
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": asset})
}

func (h *InvestmentHandler) DeleteAsset(c *gin.Context) {
	tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if err := h.deleteUC.Execute(c.Request.Context(), tenantSchema, id); err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Instrumen berhasil dihapus."})
}

func (h *InvestmentHandler) RecordUnitChange(c *gin.Context) {
	tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	assetID := c.Param("id")
	var req struct {
		TransactionType string  `json:"transaction_type" binding:"required"`
		QuantityChange  float64 `json:"quantity_change" binding:"required"`
		PricePerUnit    float64 `json:"price_per_unit" binding:"required"`
		Notes           string  `json:"notes"`
		TransactionDate string  `json:"transaction_date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_INPUT", "message": err.Error()},
		})
		return
	}

	txDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_INPUT", "message": "Format tanggal tidak valid. Gunakan YYYY-MM-DD."},
		})
		return
	}

	entry, err := h.recordUC.Execute(c.Request.Context(), usecase.RecordUnitChangeInput{
		TenantSchema:    tenantSchema,
		AssetID:         assetID,
		TransactionType: req.TransactionType,
		QuantityChange:  req.QuantityChange,
		PricePerUnit:    req.PricePerUnit,
		Notes:           req.Notes,
		TransactionDate: txDate,
	})
	if err != nil {
		h.handleDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": entry})
}

func (h *InvestmentHandler) GetUnitHistory(c *gin.Context) {
	tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	assetID := c.Param("id")
	historyMonths := parseTierIntHeader(c, headerTierHistoryMonths, tierUnlimited)

	history, err := h.historyUC.Execute(c.Request.Context(), tenantSchema, assetID, historyMonths)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	if history == nil {
		history = []entity.UnitHistory{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": history})
}
