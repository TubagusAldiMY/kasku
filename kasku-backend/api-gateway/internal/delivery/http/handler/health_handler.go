package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler menangani health check endpoint.
type HealthHandler struct {
	version string
}

// NewHealthHandler membuat instance HealthHandler baru.
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

// Health mengembalikan status kesehatan api-gateway.
// GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "api-gateway",
		"version": h.version,
	})
}
