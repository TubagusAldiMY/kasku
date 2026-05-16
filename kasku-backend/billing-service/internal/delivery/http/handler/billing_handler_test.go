package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/tests/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockHealthChecker struct {
	pgErr error
	mqErr error
}

func (m *mockHealthChecker) PingPostgres(_ context.Context) error { return m.pgErr }
func (m *mockHealthChecker) PingRabbitMQ() error                  { return m.mqErr }

func newRouter(t *testing.T, hc *mockHealthChecker) (*gin.Engine, *mocks.MockListPlansUseCase, *mocks.MockGetSubscriptionUseCase) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	listPlans := mocks.NewMockListPlansUseCase(ctrl)
	getSub := mocks.NewMockGetSubscriptionUseCase(ctrl)

	h := handler.NewBillingHandler(hc, listPlans, getSub, "1.0.0", zerolog.Nop())
	r := gin.New()
	r.GET("/health", h.Health)
	r.GET("/v1/billing/plans", h.ListPlans)
	r.GET("/v1/billing/subscription", h.GetSubscription)
	r.POST("/v1/billing/subscribe", h.Subscribe)
	r.POST("/v1/billing/webhook/midtrans", h.MidtransWebhook)
	return r, listPlans, getSub
}

func TestHealthHandler(t *testing.T) {
	t.Run("all deps ok returns 200", func(t *testing.T) {
		r, _, _ := newRouter(t, &mockHealthChecker{})
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"healthy"`)
		assert.Contains(t, w.Body.String(), `"postgres":"healthy"`)
		assert.Contains(t, w.Body.String(), `"rabbitmq":"healthy"`)
	})

	t.Run("postgres down returns 503", func(t *testing.T) {
		r, _, _ := newRouter(t, &mockHealthChecker{pgErr: errors.New("conn refused")})
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), `"postgres":"unhealthy"`)
	})

	t.Run("rabbitmq down returns 503", func(t *testing.T) {
		r, _, _ := newRouter(t, &mockHealthChecker{mqErr: errors.New("closed")})
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), `"rabbitmq":"unhealthy"`)
	})
}

func TestListPlansHandler(t *testing.T) {
	t.Run("success returns plans array", func(t *testing.T) {
		r, listPlans, _ := newRouter(t, &mockHealthChecker{})
		listPlans.EXPECT().Execute(gomock.Any()).Return([]entity.SubscriptionPlan{
			{ID: uuid.New(), Name: "FREE", PriceIDR: 0},
		}, nil)
		req := httptest.NewRequest(http.MethodGet, "/v1/billing/plans", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    []map[string]any
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Len(t, resp.Data, 1)
	})

	t.Run("usecase error returns 500", func(t *testing.T) {
		r, listPlans, _ := newRouter(t, &mockHealthChecker{})
		listPlans.EXPECT().Execute(gomock.Any()).Return(nil, errors.New("db error"))
		req := httptest.NewRequest(http.MethodGet, "/v1/billing/plans", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetSubscriptionHandler(t *testing.T) {
	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		r, _, _ := newRouter(t, &mockHealthChecker{})
		req := httptest.NewRequest(http.MethodGet, "/v1/billing/subscription", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r, _, getSub := newRouter(t, &mockHealthChecker{})
		userID := uuid.NewString()
		getSub.EXPECT().Execute(gomock.Any(), userID).Return(nil, domainerrors.ErrSubscriptionNotFound)
		req := httptest.NewRequest(http.MethodGet, "/v1/billing/subscription", nil)
		req.Header.Set("X-User-ID", userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns subscription", func(t *testing.T) {
		r, _, getSub := newRouter(t, &mockHealthChecker{})
		userID := uuid.New()
		sub := &entity.Subscription{
			ID: uuid.New(), UserID: userID, PlanID: uuid.New(), Status: entity.StatusActive,
		}
		getSub.EXPECT().Execute(gomock.Any(), userID.String()).Return(sub, nil)
		req := httptest.NewRequest(http.MethodGet, "/v1/billing/subscription", nil)
		req.Header.Set("X-User-ID", userID.String())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"ACTIVE"`)
	})
}

func TestSubscribeHandler_Placeholder(t *testing.T) {
	r, _, _ := newRouter(t, &mockHealthChecker{})
	req := httptest.NewRequest(http.MethodPost, "/v1/billing/subscribe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"COMING_SOON"`)
}

func TestMidtransWebhook_Placeholder(t *testing.T) {
	r, _, _ := newRouter(t, &mockHealthChecker{})
	req := httptest.NewRequest(http.MethodPost, "/v1/billing/webhook/midtrans", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
