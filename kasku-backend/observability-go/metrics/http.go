package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics mengembalikan Gin middleware yang mencatat:
//   - Counter  kasku_http_requests_total{service,method,route,status}
//   - Histogram kasku_http_request_duration_seconds{service,method,route}
//
// Route diambil dari c.FullPath() untuk menghindari high-cardinality dari
// path parameter (mis. /users/:id, bukan /users/abc-def-uuid).
func (r *Registry) HTTPMetrics() gin.HandlerFunc {
	labels := []string{"service", "method", "route", "status"}
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kasku_http_requests_total",
			Help: "Total HTTP request yang diterima service.",
		},
		labels,
	)

	durLabels := []string{"service", "method", "route"}
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kasku_http_request_duration_seconds",
			Help:    "Distribusi latensi HTTP per route.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		durLabels,
	)

	r.Reg.MustRegister(requests, duration)
	svc := r.Service

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// Request tidak match route (404 unrouted) — pakai "unknown" untuk
			// mencegah cardinality explosion dari path random.
			route = "unknown"
		}

		status := strconv.Itoa(c.Writer.Status())
		requests.WithLabelValues(svc, c.Request.Method, route, status).Inc()
		duration.WithLabelValues(svc, c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}
