package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	headerUserID           = "X-User-ID"
	headerTenantSchema     = "X-Tenant-Schema"
	headerSubscriptionTier = "X-Subscription-Tier"
)

// HealthChecker mendefinisikan kontrak health check untuk dependencies.
type HealthChecker interface {
	PingFinanceDB(ctx context.Context) error
	PingBillingDB(ctx context.Context) error
	PingRabbitMQ() error
}

// UserHandler menangani HTTP request untuk user management.
type UserHandler struct {
	health         HealthChecker
	serviceVersion string
	log            zerolog.Logger
}

func NewUserHandler(health HealthChecker, serviceVersion string, log zerolog.Logger) *UserHandler {
	return &UserHandler{
		health:         health,
		serviceVersion: serviceVersion,
		log:            log,
	}
}

// Health mengembalikan status kesehatan service.
func (h *UserHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()

	status := "healthy"
	httpStatus := http.StatusOK
	checks := gin.H{}

	if err := h.health.PingFinanceDB(ctx); err != nil {
		checks["finance_db"] = "unhealthy"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["finance_db"] = "healthy"
	}

	if err := h.health.PingBillingDB(ctx); err != nil {
		checks["billing_db"] = "unhealthy"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["billing_db"] = "healthy"
	}

	if err := h.health.PingRabbitMQ(); err != nil {
		checks["rabbitmq"] = "unhealthy"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["rabbitmq"] = "healthy"
	}

	c.JSON(httpStatus, gin.H{
		"status":  status,
		"version": h.serviceVersion,
		"checks":  checks,
	})
}

// GetProfile mengembalikan profil user berdasarkan JWT headers yang diinject api-gateway.
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetHeader(headerUserID)
	tenantSchema := c.GetHeader(headerTenantSchema)
	subscriptionTier := c.GetHeader(headerSubscriptionTier)

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Header autentikasi tidak ditemukan.",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":           userID,
			"tenant_schema":     tenantSchema,
			"subscription_tier": subscriptionTier,
		},
	})
}

// UpdateProfile — akan diimplementasikan di Phase 2.
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_IMPLEMENTED",
			"message": "Fitur update profil akan tersedia di versi berikutnya.",
		},
	})
}
