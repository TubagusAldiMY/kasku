package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/gin-gonic/gin"
)

func setupGin(reg *metrics.Registry) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(reg.HTTPMetrics())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	r.GET("/fail", func(c *gin.Context) { c.String(http.StatusInternalServerError, "boom") })
	r.GET("/metrics", gin.WrapH(reg.Handler()))
	return r
}

func TestHTTPMetrics_IncrementsCounterAndDuration(t *testing.T) {
	t.Parallel()
	reg := metrics.NewRegistry("test-service")
	r := setupGin(reg)

	for range 3 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(w, req)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	r.ServeHTTP(w, req)

	mw := httptest.NewRecorder()
	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(mw, mreq)
	body := mw.Body.String()

	if !strings.Contains(body, `kasku_http_requests_total{method="GET",route="/ping",service="test-service",status="200"} 3`) {
		t.Fatalf("counter /ping=3 tidak muncul. body:\n%s", body)
	}
	if !strings.Contains(body, `kasku_http_requests_total{method="GET",route="/fail",service="test-service",status="500"} 1`) {
		t.Fatalf("counter /fail=1 tidak muncul. body:\n%s", body)
	}
	if !strings.Contains(body, "kasku_http_request_duration_seconds_bucket") {
		t.Fatalf("duration histogram tidak muncul. body:\n%s", body)
	}
}

func TestHTTPMetrics_UnknownRouteCoercedToUnknown(t *testing.T) {
	t.Parallel()
	reg := metrics.NewRegistry("test-service")
	r := setupGin(reg)
	r.NoRoute(func(c *gin.Context) { c.String(http.StatusNotFound, "nope") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/random/abc-def-uuid", nil)
	r.ServeHTTP(w, req)

	mw := httptest.NewRecorder()
	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(mw, mreq)
	if !strings.Contains(mw.Body.String(), `route="unknown"`) {
		t.Fatalf("route bukan ter-collapse ke 'unknown' — risiko cardinality explosion")
	}
}

func TestBusinessCounter_DedupeOnDoubleRegister(t *testing.T) {
	t.Parallel()
	reg := metrics.NewRegistry("test-service")
	c1 := reg.Counter("kasku_test_thing_total", "test", []string{"kind"})
	c2 := reg.Counter("kasku_test_thing_total", "test", []string{"kind"})
	if c1 != c2 {
		t.Fatal("Counter(...) kedua kali harus reuse instance lama, bukan panic")
	}
	c1.WithLabelValues("a").Inc()
	c2.WithLabelValues("a").Inc()

	mw := httptest.NewRecorder()
	r := gin.New()
	r.GET("/metrics", gin.WrapH(reg.Handler()))
	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(mw, mreq)
	if !strings.Contains(mw.Body.String(), `kasku_test_thing_total{kind="a",service="test-service"} 2`) {
		t.Fatalf("counter dedupe gagal akumulasi. body:\n%s", mw.Body.String())
	}
}
