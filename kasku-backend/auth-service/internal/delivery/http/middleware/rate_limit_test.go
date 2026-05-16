package middleware_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/ratelimit"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/mocks"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupTestRouter(handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.POST("/test", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRateLimit_PanicOnInvalidConfig(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lim := mocks.NewMockLimiter(ctrl)
	log := zerolog.Nop()

	assert.Panics(t, func() {
		middleware.RateLimit(lim, middleware.RateLimitConfig{Limit: 0, Window: time.Minute, KeyFunc: middleware.KeyByClientIP, EndpointName: "x"}, log)
	}, "Limit=0 must panic")

	assert.Panics(t, func() {
		middleware.RateLimit(lim, middleware.RateLimitConfig{Limit: 10, Window: 0, KeyFunc: middleware.KeyByClientIP, EndpointName: "x"}, log)
	}, "Window=0 must panic")

	assert.Panics(t, func() {
		middleware.RateLimit(lim, middleware.RateLimitConfig{Limit: 10, Window: time.Minute, KeyFunc: nil, EndpointName: "x"}, log)
	}, "KeyFunc=nil must panic")

	assert.Panics(t, func() {
		middleware.RateLimit(lim, middleware.RateLimitConfig{Limit: 10, Window: time.Minute, KeyFunc: middleware.KeyByClientIP, EndpointName: ""}, log)
	}, "EndpointName=empty must panic")
}

func TestRateLimit_Allow(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lim := mocks.NewMockLimiter(ctrl)
	lim.EXPECT().Check(gomock.Any(), gomock.Any(), 10, time.Minute).
		Return(time.Duration(0), nil)

	r := setupTestRouter(middleware.RateLimit(lim, middleware.RateLimitConfig{
		Limit: 10, Window: time.Minute, KeyFunc: middleware.KeyByClientIP, EndpointName: "test",
	}, zerolog.Nop()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimit_Deny429(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lim := mocks.NewMockLimiter(ctrl)
	lim.EXPECT().Check(gomock.Any(), gomock.Any(), 10, time.Minute).
		Return(30*time.Second, ratelimit.ErrLimitExceeded)

	r := setupTestRouter(middleware.RateLimit(lim, middleware.RateLimitConfig{
		Limit: 10, Window: time.Minute, KeyFunc: middleware.KeyByClientIP, EndpointName: "test",
	}, zerolog.Nop()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "30", w.Header().Get("Retry-After"))
	assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))

	resetUnix, err := strconv.ParseInt(w.Header().Get("X-RateLimit-Reset"), 10, 64)
	require.NoError(t, err)
	now := time.Now().Unix()
	assert.InDelta(t, now+30, resetUnix, 2)

	body, _ := io.ReadAll(w.Body)
	assert.Contains(t, string(body), "TOO_MANY_REQUESTS")
}

func TestRateLimit_FailOpen(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lim := mocks.NewMockLimiter(ctrl)
	lim.EXPECT().Check(gomock.Any(), gomock.Any(), 10, time.Minute).
		Return(time.Duration(0), errors.New("redis down"))

	r := setupTestRouter(middleware.RateLimit(lim, middleware.RateLimitConfig{
		Limit: 10, Window: time.Minute, KeyFunc: middleware.KeyByClientIP, EndpointName: "test",
	}, zerolog.Nop()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w, req)

	// fail-open: request lewat sebagai 200, no rate-limit headers (selain warning di log)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimit_EmptyIdentityBypassed(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lim := mocks.NewMockLimiter(ctrl) // no Check expected

	keyFunc := func(c *gin.Context) string { return "" }
	r := setupTestRouter(middleware.RateLimit(lim, middleware.RateLimitConfig{
		Limit: 10, Window: time.Minute, KeyFunc: keyFunc, EndpointName: "test",
	}, zerolog.Nop()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
