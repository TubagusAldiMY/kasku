package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorrelationID_GeneratesNewWhenAbsent(t *testing.T) {
	t.Parallel()

	r := gin.New()
	var capturedID string
	r.Use(middleware.CorrelationID())
	r.GET("/x", func(c *gin.Context) {
		capturedID = middleware.GetCorrelationID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	respHeader := w.Header().Get(middleware.CorrelationIDHeader)
	assert.NotEmpty(t, respHeader)
	assert.Equal(t, respHeader, capturedID)
	_, err := uuid.Parse(respHeader)
	assert.NoError(t, err, "generated ID should be a valid UUID")
}

func TestCorrelationID_PreservesExisting(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.Use(middleware.CorrelationID())
	r.GET("/x", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	const incoming = "trace-id-abc-123"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(middleware.CorrelationIDHeader, incoming)
	r.ServeHTTP(w, req)

	assert.Equal(t, incoming, w.Header().Get(middleware.CorrelationIDHeader))
}

func TestGetCorrelationID_MissingReturnsEmpty(t *testing.T) {
	t.Parallel()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	assert.Equal(t, "", middleware.GetCorrelationID(c))
}

func TestContextWithCorrelationID_Roundtrip(t *testing.T) {
	t.Parallel()

	const id = "abc-id"
	ctx := middleware.ContextWithCorrelationID(context.Background(), id)
	got := middleware.CorrelationIDFromContext(ctx)
	require.Equal(t, id, got)
}

func TestCorrelationIDFromContext_AbsentReturnsEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", middleware.CorrelationIDFromContext(context.Background()))
}
