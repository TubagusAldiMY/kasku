package handler_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/billing-service/tests/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testWebhookSecret = "test-webhook-secret-for-unit-tests"

func init() {
	gin.SetMode(gin.TestMode)
}

type mockHealthChecker struct {
	pgErr error
	mqErr error
}

func (m *mockHealthChecker) PingPostgres(_ context.Context) error { return m.pgErr }
func (m *mockHealthChecker) PingRabbitMQ() error                  { return m.mqErr }

// mockCreatePaymentUC adalah mock sederhana untuk CreateSubscriptionPaymentUseCase.
type mockCreatePaymentUC struct {
	output *usecase.CreateSubscriptionPaymentOutput
	err    error
}

func (m *mockCreatePaymentUC) Execute(_ context.Context, _ usecase.CreateSubscriptionPaymentInput) (*usecase.CreateSubscriptionPaymentOutput, error) {
	return m.output, m.err
}

// mockHandleWebhookUC adalah mock sederhana untuk HandlePaymentWebhookUseCase.
type mockHandleWebhookUC struct {
	err error
}

func (m *mockHandleWebhookUC) Execute(_ context.Context, _ usecase.PaymentWebhookInput) error {
	return m.err
}

func newTestRouter(
	t *testing.T,
	hc *mockHealthChecker,
	createPaymentUC usecase.CreateSubscriptionPaymentUseCase,
	handleWebhookUC usecase.HandlePaymentWebhookUseCase,
) (*gin.Engine, *mocks.MockListPlansUseCase, *mocks.MockGetSubscriptionUseCase) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	if createPaymentUC == nil {
		createPaymentUC = &mockCreatePaymentUC{}
	}
	if handleWebhookUC == nil {
		handleWebhookUC = &mockHandleWebhookUC{}
	}

	listPlans := mocks.NewMockListPlansUseCase(ctrl)
	getSub := mocks.NewMockGetSubscriptionUseCase(ctrl)

	h := handler.NewBillingHandler(
		hc,
		listPlans,
		getSub,
		createPaymentUC,
		handleWebhookUC,
		testWebhookSecret,
		"1.0.0-test",
		zerolog.Nop(),
	)

	r := gin.New()
	r.GET("/health", h.Health)
	r.GET("/v1/billing/plans", h.ListPlans)
	r.GET("/v1/billing/subscription", h.GetSubscription)
	r.POST("/v1/billing/subscribe", h.Subscribe)
	r.POST("/v1/billing/webhook/payment", h.PaymentWebhook)
	return r, listPlans, getSub
}

// computeTestSignature menghitung HMAC-SHA256 untuk body webhook dalam test.
func computeTestSignature(body []byte) string {
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestHealthHandler(t *testing.T) {
	t.Run("all deps ok returns 200", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"healthy"`)
		assert.Contains(t, w.Body.String(), `"postgres":"healthy"`)
		assert.Contains(t, w.Body.String(), `"rabbitmq":"healthy"`)
	})

	t.Run("postgres down returns 503", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{pgErr: errors.New("conn refused")}, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), `"postgres":"unhealthy"`)
	})

	t.Run("rabbitmq down returns 503", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{mqErr: errors.New("closed")}, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), `"rabbitmq":"unhealthy"`)
	})
}

func TestListPlansHandler(t *testing.T) {
	t.Run("success returns plans array", func(t *testing.T) {
		r, listPlans, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
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
		r, listPlans, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		listPlans.EXPECT().Execute(gomock.Any()).Return(nil, errors.New("db error"))
		req := httptest.NewRequest(http.MethodGet, "/v1/billing/plans", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetSubscriptionHandler(t *testing.T) {
	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		req := httptest.NewRequest(http.MethodGet, "/v1/billing/subscription", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r, _, getSub := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		userID := uuid.NewString()
		getSub.EXPECT().Execute(gomock.Any(), userID).Return(nil, domainerrors.ErrSubscriptionNotFound)
		req := httptest.NewRequest(http.MethodGet, "/v1/billing/subscription", nil)
		req.Header.Set("X-User-ID", userID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns subscription", func(t *testing.T) {
		r, _, getSub := newTestRouter(t, &mockHealthChecker{}, nil, nil)
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

func TestSubscribeHandler(t *testing.T) {
	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		body := `{"plan_id":"` + uuid.NewString() + `"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/subscribe", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing plan_id returns 400", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/subscribe", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid payment method returns 400", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		body := `{"plan_id":"` + uuid.NewString() + `","payment_method":"BANK_TRANSFER"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/subscribe", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("gateway unavailable returns 503", func(t *testing.T) {
		createUC := &mockCreatePaymentUC{err: domainerrors.ErrPaymentGatewayUnavailable}
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, createUC, nil)
		body := `{"plan_id":"` + uuid.NewString() + `"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/subscribe", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "PAYMENT_GATEWAY_UNAVAILABLE")
	})

	t.Run("success returns payment info", func(t *testing.T) {
		expiresAt := time.Now().Add(time.Hour)
		createUC := &mockCreatePaymentUC{
			output: &usecase.CreateSubscriptionPaymentOutput{
				PaymentID:  uuid.New(),
				OrderID:    "KASKU-SUB-test-123",
				AmountIDR:  49000,
				PaymentURL: "https://payment.example.com/qr/abc",
				ExpiresAt:  &expiresAt,
			},
		}
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, createUC, nil)
		body := `{"plan_id":"` + uuid.NewString() + `","payment_method":"QRIS"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/subscribe", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"order_id":"KASKU-SUB-test-123"`)
		assert.Contains(t, w.Body.String(), `"amount_idr":49000`)
	})
}

func TestPaymentWebhookHandler(t *testing.T) {
	buildWebhookPayload := func() []byte {
		payload := map[string]any{
			"type":            "deposit",
			"provider_trx_id": uuid.NewString(),
			"ref_id":          "KASKU-SUB-" + uuid.NewString() + "-1234567890",
			"amount":          49000,
			"status":          "success",
		}
		data, _ := json.Marshal(payload)
		return data
	}

	t.Run("missing signature returns 400", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		body := buildWebhookPayload()
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/webhook/payment", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// Tidak ada X-Signature
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "INVALID_SIGNATURE")
	})

	t.Run("wrong signature returns 400", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		body := buildWebhookPayload()
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/webhook/payment", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", "wrong-signature-hex")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid signature and success webhook returns 200", func(t *testing.T) {
		handleUC := &mockHandleWebhookUC{err: nil}
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, handleUC)
		body := buildWebhookPayload()
		sig := computeTestSignature(body)
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/webhook/payment", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", sig)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
	})

	t.Run("usecase error returns 500 so orchestrator retries", func(t *testing.T) {
		handleUC := &mockHandleWebhookUC{err: errors.New("db connection lost")}
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, handleUC)
		body := buildWebhookPayload()
		sig := computeTestSignature(body)
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/webhook/payment", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", sig)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		r, _, _ := newTestRouter(t, &mockHealthChecker{}, nil, nil)
		// Payload valid JSON tapi tidak ada field ref_id
		body := []byte(`{"provider_trx_id":"abc","status":"success"}`)
		sig := computeTestSignature(body)
		req := httptest.NewRequest(http.MethodPost, "/v1/billing/webhook/payment", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Signature", sig)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
