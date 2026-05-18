package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/handler"
	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/mocks"
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

type testDeps struct {
	register *mocks.MockRegisterUseCase
	verify   *mocks.MockVerifyEmailUseCase
	resend   *mocks.MockResendVerificationUseCase
	login    *mocks.MockLoginUseCase
	refresh  *mocks.MockRefreshTokenUseCase
	logout   *mocks.MockLogoutUseCase
	forgot   *mocks.MockForgotPasswordUseCase
	reset    *mocks.MockResetPasswordUseCase
	health   *mockHealthChecker
}

// mockHealthChecker simple stub. HealthChecker interface dari handler package
// — pakai struct stub, bukan gomock, supaya test concise.
type mockHealthChecker struct {
	pgErr, redisErr, mqErr error
}

func (m *mockHealthChecker) PingPostgres(_ context.Context) error { return m.pgErr }
func (m *mockHealthChecker) PingRedis(_ context.Context) error    { return m.redisErr }
func (m *mockHealthChecker) PingRabbitMQ() error                  { return m.mqErr }

func newDeps(t *testing.T) (*testDeps, *gin.Engine) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	d := &testDeps{
		register: mocks.NewMockRegisterUseCase(ctrl),
		verify:   mocks.NewMockVerifyEmailUseCase(ctrl),
		resend:   mocks.NewMockResendVerificationUseCase(ctrl),
		login:    mocks.NewMockLoginUseCase(ctrl),
		refresh:  mocks.NewMockRefreshTokenUseCase(ctrl),
		logout:   mocks.NewMockLogoutUseCase(ctrl),
		forgot:   mocks.NewMockForgotPasswordUseCase(ctrl),
		reset:    mocks.NewMockResetPasswordUseCase(ctrl),
		health:   &mockHealthChecker{},
	}
	h := handler.NewAuthHandler(
		d.register, d.verify, d.resend, d.login, d.refresh, d.logout, d.forgot, d.reset,
		d.health, "1.0.0", true, zerolog.Nop(),
	)
	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/verify-email", h.VerifyEmail)
	r.POST("/auth/resend-verification", h.ResendVerification)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/forgot-password", h.ForgotPassword)
	r.POST("/auth/reset-password", h.ResetPassword)
	r.GET("/health", h.Health)
	return d, r
}

func doJSON(r *gin.Engine, method, path string, body any, headers map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ─── Register ────────────────────────────────────────────────────────────────

func TestAuthHandler_Register_Happy(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)

	uid := uuid.New()
	d.register.EXPECT().Execute(gomock.Any(), usecase.RegisterInput{
		Email: "u@ex.com", Username: "alice", Password: "Pass1234",
	}).Return(&usecase.RegisterOutput{UserID: uid, Email: "u@ex.com", Username: "alice"}, nil)

	w := doJSON(r, "POST", "/auth/register", gin.H{
		"email": "u@ex.com", "username": "alice", "password": "Pass1234",
	}, nil)

	require.Equal(t, http.StatusCreated, w.Code)
	body, _ := io.ReadAll(w.Body)
	assert.Contains(t, string(body), uid.String())
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	t.Parallel()
	_, r := newDeps(t)
	w := doJSON(r, "POST", "/auth/register", gin.H{"email": ""}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_DomainError(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.register.EXPECT().Execute(gomock.Any(), gomock.Any()).
		Return(nil, domainerrors.ErrEmailAlreadyExists)

	w := doJSON(r, "POST", "/auth/register", gin.H{
		"email": "dup@ex.com", "username": "x", "password": "Pass1234",
	}, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ─── VerifyEmail ─────────────────────────────────────────────────────────────

func TestAuthHandler_VerifyEmail_Happy(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.verify.EXPECT().Execute(gomock.Any(), "raw-token").Return(nil)

	w := doJSON(r, "POST", "/auth/verify-email?token=raw-token", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_VerifyEmail_MissingToken(t *testing.T) {
	t.Parallel()
	_, r := newDeps(t)
	w := doJSON(r, "POST", "/auth/verify-email", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_VerifyEmail_InvalidToken(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.verify.EXPECT().Execute(gomock.Any(), "bad").Return(domainerrors.ErrInvalidToken)
	w := doJSON(r, "POST", "/auth/verify-email?token=bad", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── ResendVerification ──────────────────────────────────────────────────────

func TestAuthHandler_ResendVerification_AlwaysOK(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.resend.EXPECT().Execute(gomock.Any(), "any@ex.com").Return(nil)
	w := doJSON(r, "POST", "/auth/resend-verification", gin.H{"email": "any@ex.com"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ResendVerification_ValidationError(t *testing.T) {
	t.Parallel()
	_, r := newDeps(t)
	w := doJSON(r, "POST", "/auth/resend-verification", gin.H{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestAuthHandler_Login_HappySetsRefreshCookie(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.login.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&usecase.LoginOutput{
		AccessToken: "jwt-token",
		TokenType:   "Bearer",
		ExpiresIn:   900,
		RefreshTokenCookie: usecase.RefreshTokenCookieParams{
			RawToken: "refresh-raw",
			MaxAge:   86400,
			IsSecure: false,
		},
	}, nil)

	w := doJSON(r, "POST", "/auth/login", gin.H{"email": "u@ex.com", "password": "P1"}, nil)
	require.Equal(t, http.StatusOK, w.Code)

	cookie := w.Result().Cookies()
	require.Len(t, cookie, 1)
	assert.Equal(t, "refresh_token", cookie[0].Name)
	assert.Equal(t, "refresh-raw", cookie[0].Value)
	assert.True(t, cookie[0].HttpOnly)
	assert.Equal(t, "/v1/auth", cookie[0].Path)
}

func TestAuthHandler_Login_InvalidCreds(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.login.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, domainerrors.ErrInvalidCredentials)
	w := doJSON(r, "POST", "/auth/login", gin.H{"email": "u@ex.com", "password": "x"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Login_AccountLocked(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.login.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, domainerrors.ErrAccountLocked)
	w := doJSON(r, "POST", "/auth/login", gin.H{"email": "u@ex.com", "password": "x"}, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuthHandler_Login_ValidationError(t *testing.T) {
	t.Parallel()
	_, r := newDeps(t)
	w := doJSON(r, "POST", "/auth/login", gin.H{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestAuthHandler_Refresh_Happy(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.refresh.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&usecase.LoginOutput{
		AccessToken: "jwt2", TokenType: "Bearer", ExpiresIn: 900,
		RefreshTokenCookie: usecase.RefreshTokenCookieParams{RawToken: "new-refresh", MaxAge: 86400},
	}, nil)

	w := doJSON(r, "POST", "/auth/refresh", nil, nil,
		&http.Cookie{Name: "refresh_token", Value: "old-raw"})
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, w.Result().Cookies(), 1)
	assert.Equal(t, "new-refresh", w.Result().Cookies()[0].Value)
}

func TestAuthHandler_Refresh_NoCookie(t *testing.T) {
	t.Parallel()
	_, r := newDeps(t)
	w := doJSON(r, "POST", "/auth/refresh", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.refresh.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, domainerrors.ErrInvalidToken)
	w := doJSON(r, "POST", "/auth/refresh", nil, nil,
		&http.Cookie{Name: "refresh_token", Value: "bad"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout_ClearsCookie(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.logout.EXPECT().Execute(gomock.Any(), usecase.LogoutInput{
		AccessToken:     "jwt-abc",
		RawRefreshToken: "refresh-abc",
	}).Return(nil)

	w := doJSON(r, "POST", "/auth/logout", nil,
		map[string]string{"Authorization": "Bearer jwt-abc"},
		&http.Cookie{Name: "refresh_token", Value: "refresh-abc"})
	require.Equal(t, http.StatusOK, w.Code)

	// MaxAge=-1 → cookie cleared
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "", cookies[0].Value)
	assert.LessOrEqual(t, cookies[0].MaxAge, 0)
}

// ─── ForgotPassword ──────────────────────────────────────────────────────────

func TestAuthHandler_ForgotPassword_AlwaysOK(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.forgot.EXPECT().Execute(gomock.Any(), "u@ex.com").Return(nil)
	w := doJSON(r, "POST", "/auth/forgot-password", gin.H{"email": "u@ex.com"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ForgotPassword_ValidationError(t *testing.T) {
	t.Parallel()
	_, r := newDeps(t)
	w := doJSON(r, "POST", "/auth/forgot-password", gin.H{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ResetPassword ───────────────────────────────────────────────────────────

func TestAuthHandler_ResetPassword_Happy(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.reset.EXPECT().Execute(gomock.Any(), "raw-tok", "NewPass123").Return(nil)
	w := doJSON(r, "POST", "/auth/reset-password",
		gin.H{"token": "raw-tok", "new_password": "NewPass123"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ResetPassword_InvalidToken(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.reset.EXPECT().Execute(gomock.Any(), "bad", "NewPass123").Return(domainerrors.ErrInvalidToken)
	w := doJSON(r, "POST", "/auth/reset-password",
		gin.H{"token": "bad", "new_password": "NewPass123"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_ResetPassword_GenericErrorMaps500(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.reset.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("non-domain error"))
	w := doJSON(r, "POST", "/auth/reset-password",
		gin.H{"token": "x", "new_password": "Y123"}, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthHandler_ResetPassword_ValidationError(t *testing.T) {
	t.Parallel()
	_, r := newDeps(t)
	w := doJSON(r, "POST", "/auth/reset-password", gin.H{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Health ──────────────────────────────────────────────────────────────────

func TestAuthHandler_Health_AllHealthy(t *testing.T) {
	t.Parallel()
	_, r := newDeps(t)
	w := doJSON(r, "GET", "/health", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body, _ := io.ReadAll(w.Body)
	assert.Contains(t, string(body), `"status":"healthy"`)
	assert.Contains(t, string(body), `"postgres"`)
	assert.Contains(t, string(body), `"redis"`)
	assert.Contains(t, string(body), `"rabbitmq"`)
}

func TestAuthHandler_Health_Degraded(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.health.pgErr = fmt.Errorf("pg down")
	w := doJSON(r, "GET", "/health", nil, nil)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	body, _ := io.ReadAll(w.Body)
	assert.Contains(t, string(body), `"status":"unhealthy"`)
}

func TestAuthHandler_Health_AllUnhealthy(t *testing.T) {
	t.Parallel()
	d, r := newDeps(t)
	d.health.pgErr = fmt.Errorf("pg")
	d.health.redisErr = fmt.Errorf("redis")
	d.health.mqErr = fmt.Errorf("mq")
	w := doJSON(r, "GET", "/health", nil, nil)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
