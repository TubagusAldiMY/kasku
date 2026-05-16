package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	serviceVersion string
}

func NewHealthHandler(serviceVersion string) *HealthHandler {
	return &HealthHandler{serviceVersion: serviceVersion}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":  "healthy",
			"service": "admin-service",
			"version": h.serviceVersion,
		},
	})
}
