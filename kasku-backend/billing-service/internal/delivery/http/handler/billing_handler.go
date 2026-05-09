package handler

import (
	"context"
	"net/http"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	// headerUserID adalah nama header yang di-inject oleh api-gateway setelah verifikasi JWT.
	headerUserID = "X-User-ID"
)

// HealthChecker mendefinisikan kontrak untuk health check dependencies.
type HealthChecker interface {
	PingPostgres(ctx context.Context) error
}

// PlansLister mendefinisikan kontrak use case untuk mengambil daftar plan.
type PlansLister interface {
	Execute(ctx context.Context) ([]entity.SubscriptionPlan, error)
}

// SubscriptionGetter mendefinisikan kontrak use case untuk mengambil subscription user.
type SubscriptionGetter interface {
	Execute(ctx context.Context, userID string) (*entity.Subscription, error)
}

// BillingHandler menangani HTTP request untuk endpoint billing.
// Business logic tidak boleh ada di sini — semua didelegasikan ke use case.
type BillingHandler struct {
	health            HealthChecker
	listPlansUC       PlansLister
	getSubscriptionUC SubscriptionGetter
	serviceVersion    string
	log               zerolog.Logger
}

// NewBillingHandler membuat instance BillingHandler baru dengan semua dependensi yang diinjeksikan.
func NewBillingHandler(
	health HealthChecker,
	listPlansUC PlansLister,
	getSubscriptionUC SubscriptionGetter,
	serviceVersion string,
	log zerolog.Logger,
) *BillingHandler {
	return &BillingHandler{
		health:            health,
		listPlansUC:       listPlansUC,
		getSubscriptionUC: getSubscriptionUC,
		serviceVersion:    serviceVersion,
		log:               log,
	}
}

// Health mengembalikan status kesehatan service beserta status masing-masing dependency.
// HTTP 200 = healthy, HTTP 503 = degraded (salah satu dependency bermasalah).
func (h *BillingHandler) Health(c *gin.Context) {
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

// planResponse adalah DTO untuk respons daftar plan — memisahkan domain entity dari transport layer.
type planResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	PriceIDR int               `json:"price_idr"`
	Limits   entity.PlanLimits `json:"limits"`
}

// ListPlans mengembalikan semua subscription plan yang aktif.
// Endpoint ini public (tidak memerlukan autentikasi) untuk keperluan halaman pricing.
func (h *BillingHandler) ListPlans(c *gin.Context) {
	plans, err := h.listPlansUC.Execute(c.Request.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("gagal mengambil daftar subscription plan")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Gagal mengambil daftar plan."},
		})
		return
	}

	result := make([]planResponse, 0, len(plans))
	for _, p := range plans {
		result = append(result, planResponse{
			ID:       p.ID.String(),
			Name:     p.Name,
			PriceIDR: p.PriceIDR,
			Limits:   p.Limits,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// GetSubscription mengembalikan detail subscription aktif milik user yang terautentikasi.
// Memerlukan header X-User-ID yang di-inject oleh api-gateway.
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	userID := c.GetHeader(headerUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Header autentikasi tidak ditemukan."},
		})
		return
	}

	sub, err := h.getSubscriptionUC.Execute(c.Request.Context(), userID)
	if err != nil {
		if domainerrors.IsDomainError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   gin.H{"code": "SUBSCRIPTION_NOT_FOUND", "message": "Subscription tidak ditemukan."},
			})
			return
		}
		correlationID, _ := c.Get("correlation_id")
		h.log.Error().
			Err(err).
			Str("user_id", userID).
			Interface("correlation_id", correlationID).
			Msg("gagal mengambil subscription")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Gagal mengambil subscription."},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                   sub.ID.String(),
			"user_id":              sub.UserID.String(),
			"plan_id":              sub.PlanID.String(),
			"status":               string(sub.Status),
			"current_period_start": sub.CurrentPeriodStart,
			"current_period_end":   sub.CurrentPeriodEnd,
		},
	})
}

// Subscribe adalah placeholder untuk endpoint langganan berbayar.
// Implementasi penuh (Midtrans Snap API) akan dikerjakan di Phase 2.
func (h *BillingHandler) Subscribe(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "COMING_SOON",
			"message": "Fitur pembayaran akan segera hadir.",
		},
	})
}

// MidtransWebhook menerima notifikasi payment dari Midtrans.
// Phase 1: hanya acknowledge (HTTP 200). Phase 2 akan memproses idempotency dan update subscription.
func (h *BillingHandler) MidtransWebhook(c *gin.Context) {
	// Catatan Phase 2: verifikasi SHA512(order_id+status_code+gross_amount+SERVER_KEY)
	// sebelum memproses apapun, lalu lakukan idempotency check via order_id.
	c.JSON(http.StatusOK, gin.H{"success": true})
}
