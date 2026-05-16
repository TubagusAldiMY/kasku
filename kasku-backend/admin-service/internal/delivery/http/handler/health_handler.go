package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthChecker mengumpulkan dependency yang di-probe pada endpoint /health.
type HealthChecker struct {
	AdminPool   *pgxpool.Pool
	AuthPool    *pgxpool.Pool
	BillingPool *pgxpool.Pool
	RedisClient *redis.Client
}

// HealthHandler menulis status liveness + dependency check.
type HealthHandler struct {
	serviceVersion string
	checker        *HealthChecker
}

// NewHealthHandler membuat handler yang mengecek 3 DB pool + Redis.
func NewHealthHandler(serviceVersion string, checker *HealthChecker) *HealthHandler {
	return &HealthHandler{serviceVersion: serviceVersion, checker: checker}
}

// Health menulis status overall + per-dependency.
// Mengembalikan HTTP 503 (degraded) bila ada dependency unhealthy.
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{}
	status := "healthy"
	httpStatus := http.StatusOK

	if h.checker != nil {
		if h.checker.AdminPool != nil {
			if err := h.checker.AdminPool.Ping(ctx); err != nil {
				checks["kasku_admin"] = "unhealthy"
				status = "degraded"
				httpStatus = http.StatusServiceUnavailable
			} else {
				checks["kasku_admin"] = "healthy"
			}
		}
		if h.checker.AuthPool != nil {
			if err := h.checker.AuthPool.Ping(ctx); err != nil {
				checks["kasku_auth"] = "unhealthy"
				status = "degraded"
				httpStatus = http.StatusServiceUnavailable
			} else {
				checks["kasku_auth"] = "healthy"
			}
		}
		if h.checker.BillingPool != nil {
			if err := h.checker.BillingPool.Ping(ctx); err != nil {
				checks["kasku_billing"] = "unhealthy"
				status = "degraded"
				httpStatus = http.StatusServiceUnavailable
			} else {
				checks["kasku_billing"] = "healthy"
			}
		}
		if h.checker.RedisClient != nil {
			if err := h.checker.RedisClient.Ping(ctx).Err(); err != nil {
				checks["redis"] = "unhealthy"
				status = "degraded"
				httpStatus = http.StatusServiceUnavailable
			} else {
				checks["redis"] = "healthy"
			}
		}
	}

	c.JSON(httpStatus, gin.H{
		"success": status == "healthy",
		"data": gin.H{
			"status":  status,
			"service": "admin-service",
			"version": h.serviceVersion,
			"checks":  checks,
		},
	})
}
