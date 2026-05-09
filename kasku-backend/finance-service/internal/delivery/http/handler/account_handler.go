package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	headerUserID            = "X-User-ID"
	headerTenantSchema      = "X-Tenant-Schema"
	headerTierMaxAccounts   = "X-Tier-Max-Accounts"
	headerTierHistoryMonths = "X-Tier-History-Months"

	tierUnlimited = -1
)

// HealthChecker mendefinisikan kontrak untuk health check dependency.
type HealthChecker interface {
	PingPostgres(ctx context.Context) error
}

type AccountHandler struct {
	createUC       *usecase.CreateAccountUseCase
	listUC         *usecase.ListAccountsUseCase
	getUC          *usecase.GetAccountUseCase
	updateUC       *usecase.UpdateAccountUseCase
	deleteUC       *usecase.DeleteAccountUseCase
	historyUC      *usecase.GetBalanceHistoryUseCase
	health         HealthChecker
	serviceVersion string
	log            zerolog.Logger
}

func NewAccountHandler(
	createUC *usecase.CreateAccountUseCase,
	listUC *usecase.ListAccountsUseCase,
	getUC *usecase.GetAccountUseCase,
	updateUC *usecase.UpdateAccountUseCase,
	deleteUC *usecase.DeleteAccountUseCase,
	historyUC *usecase.GetBalanceHistoryUseCase,
	health HealthChecker,
	serviceVersion string,
	log zerolog.Logger,
) *AccountHandler {
	return &AccountHandler{
		createUC:       createUC,
		listUC:         listUC,
		getUC:          getUC,
		updateUC:       updateUC,
		deleteUC:       deleteUC,
		historyUC:      historyUC,
		health:         health,
		serviceVersion: serviceVersion,
		log:            log,
	}
}

func (h *AccountHandler) Health(c *gin.Context) {
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
		"version": h.serviceVersion,
		"checks":  checks,
	})
}

// extractRequestContext mengekstrak dan memvalidasi header autentikasi dari api-gateway.
// Mengembalikan false dan menulis response error jika header tidak lengkap.
func (h *AccountHandler) extractRequestContext(c *gin.Context) (userID, tenantSchema string, ok bool) {
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

// parseTierIntHeader membaca header tier integer; mengembalikan defaultValue jika header kosong atau tidak valid.
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

// handleDomainError memetakan domain error ke HTTP status yang sesuai.
func (h *AccountHandler) handleDomainError(c *gin.Context, err error, correlationID string) {
	if de, ok := domainerrors.IsDomainError(err); ok {
		switch de.Code {
		case domainerrors.ErrAccountNotFound.Code:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		case domainerrors.ErrAccountLimitReached.Code:
			// HTTP 402 sesuai konvensi KasKu untuk tier limit exceeded
			c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": de.Code, "message": de.Message}})
		}
		return
	}
	h.log.Error().Err(err).Str("correlation_id", correlationID).Msg("unexpected internal error")
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Terjadi kesalahan internal."},
	})
}

func (h *AccountHandler) ListAccounts(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	accounts, err := h.listUC.Execute(c.Request.Context(), tenantSchema, userID)
	if err != nil {
		correlationID, _ := c.Get("correlation_id")
		h.handleDomainError(c, err, correlationID.(string))
		return
	}

	// Kembalikan slice kosong, bukan null, untuk konsistensi response API
	if accounts == nil {
		accounts = []entity.FinancialAccount{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": accounts})
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	var req struct {
		Name        string             `json:"name" binding:"required"`
		AccountType entity.AccountType `json:"account_type" binding:"required"`
		Balance     int64              `json:"balance"`
		Currency    string             `json:"currency"`
		Color       string             `json:"color"`
		Icon        string             `json:"icon"`
		IsDefault   bool               `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_INPUT", "message": err.Error()},
		})
		return
	}

	maxAccounts := parseTierIntHeader(c, headerTierMaxAccounts, tierUnlimited)

	account, err := h.createUC.Execute(c.Request.Context(), usecase.CreateAccountInput{
		TenantSchema: tenantSchema,
		UserID:       userID,
		Name:         req.Name,
		AccountType:  req.AccountType,
		Balance:      req.Balance,
		Currency:     req.Currency,
		Color:        req.Color,
		Icon:         req.Icon,
		IsDefault:    req.IsDefault,
		MaxAccounts:  maxAccounts,
	})
	if err != nil {
		correlationID, _ := c.Get("correlation_id")
		h.handleDomainError(c, err, correlationID.(string))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": account})
}

func (h *AccountHandler) GetAccount(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	account, err := h.getUC.Execute(c.Request.Context(), tenantSchema, id, userID)
	if err != nil {
		correlationID, _ := c.Get("correlation_id")
		h.handleDomainError(c, err, correlationID.(string))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": account})
}

func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	var req struct {
		Name        string             `json:"name" binding:"required"`
		AccountType entity.AccountType `json:"account_type" binding:"required"`
		Color       string             `json:"color"`
		Icon        string             `json:"icon"`
		IsDefault   bool               `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_INPUT", "message": err.Error()},
		})
		return
	}

	account, err := h.updateUC.Execute(c.Request.Context(), usecase.UpdateAccountInput{
		TenantSchema: tenantSchema,
		ID:           id,
		UserID:       userID,
		Name:         req.Name,
		AccountType:  req.AccountType,
		Color:        req.Color,
		Icon:         req.Icon,
		IsDefault:    req.IsDefault,
	})
	if err != nil {
		correlationID, _ := c.Get("correlation_id")
		h.handleDomainError(c, err, correlationID.(string))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": account})
}

func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if err := h.deleteUC.Execute(c.Request.Context(), tenantSchema, id, userID); err != nil {
		correlationID, _ := c.Get("correlation_id")
		h.handleDomainError(c, err, correlationID.(string))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Akun berhasil dihapus."})
}

func (h *AccountHandler) GetBalanceHistory(c *gin.Context) {
	userID, tenantSchema, ok := h.extractRequestContext(c)
	if !ok {
		return
	}

	accountID := c.Param("id")
	historyMonths := parseTierIntHeader(c, headerTierHistoryMonths, tierUnlimited)

	history, err := h.historyUC.Execute(c.Request.Context(), tenantSchema, accountID, userID, historyMonths)
	if err != nil {
		correlationID, _ := c.Get("correlation_id")
		h.handleDomainError(c, err, correlationID.(string))
		return
	}

	if history == nil {
		history = []entity.BalanceHistory{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": history})
}
