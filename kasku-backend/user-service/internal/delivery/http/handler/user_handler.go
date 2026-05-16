package handler

import (
	"context"
	"net/http"

	"github.com/TubagusAldiMY/kasku/user-service/internal/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	headerUserID           = "X-User-ID"
	headerUserEmail        = "X-User-Email"
	headerTenantSchema     = "X-Tenant-Schema"
	headerSubscriptionTier = "X-Subscription-Tier"
)

// HealthChecker mendefinisikan kontrak health check untuk dependencies.
type HealthChecker interface {
	PingFinanceDB(ctx context.Context) error
	PingBillingDB(ctx context.Context) error
	PingUserDB(ctx context.Context) error
	PingRabbitMQ() error
}

// UserHandler menangani HTTP request untuk user management.
type UserHandler struct {
	health         HealthChecker
	profileRepo    persistence.UserProfileRepository
	serviceVersion string
	log            zerolog.Logger
}

func NewUserHandler(health HealthChecker, profileRepo persistence.UserProfileRepository, serviceVersion string, log zerolog.Logger) *UserHandler {
	return &UserHandler{
		health:         health,
		profileRepo:    profileRepo,
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

	if err := h.health.PingUserDB(ctx); err != nil {
		checks["user_db"] = "unhealthy"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["user_db"] = "healthy"
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
	email := c.GetHeader(headerUserEmail)
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

	profile, err := h.profileRepo.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Str("user_id", userID).Msg("gagal get profile")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Gagal mengambil profil."},
		})
		return
	}

	if profile != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"user_id":           profile.UserID,
				"email":             profile.Email,
				"username":          profile.Username,
				"display_name":      profile.DisplayName,
				"tenant_schema":     tenantSchema,
				"subscription_tier": subscriptionTier,
				"created_at":        profile.CreatedAt,
				"updated_at":        profile.UpdatedAt,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":           userID,
			"email":             email,
			"username":          "",
			"display_name":      nil,
			"tenant_schema":     tenantSchema,
			"subscription_tier": subscriptionTier,
		},
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetHeader(headerUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Header autentikasi tidak ditemukan."},
		})
		return
	}

	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_INPUT", "message": "Format request tidak valid."},
		})
		return
	}

	profile, err := h.profileRepo.UpdateUserProfile(c.Request.Context(), userID, req.Username, req.DisplayName)
	if err != nil {
		h.log.Error().Err(err).Str("user_id", userID).Msg("gagal update profile")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "PROFILE_UPDATE_FAILED", "message": "Gagal memperbarui profil."},
		})
		return
	}
	if profile == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   gin.H{"code": "PROFILE_NOT_FOUND", "message": "Profil belum tersedia."},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}
