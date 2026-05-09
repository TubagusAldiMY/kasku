package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RabbitMQPinger mendefinisikan kontrak health check untuk RabbitMQ consumer.
type RabbitMQPinger interface {
	Ping() error
}

// HealthHandler menangani endpoint health check service.
type HealthHandler struct {
	consumer       RabbitMQPinger
	serviceVersion string
}

func NewHealthHandler(consumer RabbitMQPinger, serviceVersion string) *HealthHandler {
	return &HealthHandler{consumer: consumer, serviceVersion: serviceVersion}
}

// Health mengembalikan status kesehatan service beserta dependency check.
// HTTP 200 jika healthy, HTTP 503 jika ada dependency yang degraded.
func (h *HealthHandler) Health(c *gin.Context) {
	status := "healthy"
	httpStatus := http.StatusOK
	checks := gin.H{}

	if err := h.consumer.Ping(); err != nil {
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
