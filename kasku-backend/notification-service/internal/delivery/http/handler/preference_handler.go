package handler

import (
	"net/http"

	"github.com/TubagusAldiMY/kasku/notification-service/internal/infrastructure/persistence"
	"github.com/gin-gonic/gin"
)

const headerUserID = "X-User-ID"

type PreferenceHandler struct {
	repo persistence.PreferenceRepository
}

func NewPreferenceHandler(repo persistence.PreferenceRepository) *PreferenceHandler {
	return &PreferenceHandler{repo: repo}
}

func (h *PreferenceHandler) Get(c *gin.Context) {
	userID := c.GetHeader(headerUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Header autentikasi tidak ditemukan."},
		})
		return
	}

	pref, err := h.repo.Get(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Gagal mengambil preference."},
		})
		return
	}
	if pref == nil {
		pref = defaultPreference()
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": pref})
}

func (h *PreferenceHandler) Update(c *gin.Context) {
	userID := c.GetHeader(headerUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Header autentikasi tidak ditemukan."},
		})
		return
	}

	var req persistence.NotificationPreference
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_INPUT", "message": "Format request tidak valid."},
		})
		return
	}

	pref, err := h.repo.Upsert(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Gagal menyimpan preference."},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": pref})
}

func defaultPreference() *persistence.NotificationPreference {
	return &persistence.NotificationPreference{
		EmailEnabled:         true,
		PaymentAlertsEnabled: true,
		ExpiryAlertsEnabled:  true,
	}
}
