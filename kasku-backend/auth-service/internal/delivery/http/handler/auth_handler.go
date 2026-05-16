package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/dto"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/response"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const (
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/auth/refresh"
)

// HealthChecker mendefinisikan kontrak untuk dependency health check.
type HealthChecker interface {
	PingPostgres(ctx context.Context) error
	PingRedis(ctx context.Context) error
	PingRabbitMQ() error
}

// AuthHandler menangani semua HTTP request ke endpoint /auth/*.
// Handler hanya bertanggung jawab untuk:
// 1. Parsing dan validasi format request
// 2. Memanggil use case yang sesuai
// 3. Memetakan hasil ke HTTP response
type AuthHandler struct {
	registerUC           usecase.RegisterUseCase
	verifyEmailUC        usecase.VerifyEmailUseCase
	resendVerificationUC usecase.ResendVerificationUseCase
	loginUC              usecase.LoginUseCase
	refreshTokenUC       usecase.RefreshTokenUseCase
	logoutUC             usecase.LogoutUseCase
	forgotPasswordUC     usecase.ForgotPasswordUseCase
	resetPasswordUC      usecase.ResetPasswordUseCase
	healthChecker        HealthChecker
	serviceVersion       string
	isDev                bool
	logger               zerolog.Logger
}

// NewAuthHandler membuat instance AuthHandler dengan semua dependency.
func NewAuthHandler(
	registerUC usecase.RegisterUseCase,
	verifyEmailUC usecase.VerifyEmailUseCase,
	resendVerificationUC usecase.ResendVerificationUseCase,
	loginUC usecase.LoginUseCase,
	refreshTokenUC usecase.RefreshTokenUseCase,
	logoutUC usecase.LogoutUseCase,
	forgotPasswordUC usecase.ForgotPasswordUseCase,
	resetPasswordUC usecase.ResetPasswordUseCase,
	healthChecker HealthChecker,
	serviceVersion string,
	isDev bool,
	logger zerolog.Logger,
) *AuthHandler {
	return &AuthHandler{
		registerUC:           registerUC,
		verifyEmailUC:        verifyEmailUC,
		resendVerificationUC: resendVerificationUC,
		loginUC:              loginUC,
		refreshTokenUC:       refreshTokenUC,
		logoutUC:             logoutUC,
		forgotPasswordUC:     forgotPasswordUC,
		resetPasswordUC:      resetPasswordUC,
		healthChecker:        healthChecker,
		serviceVersion:       serviceVersion,
		isDev:                isDev,
		logger:               logger,
	}
}

// Register menangani POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Format request tidak valid.", err.Error())
		return
	}

	out, err := h.registerUC.Execute(c.Request.Context(), usecase.RegisterInput{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		h.logError(c, "register", err)
		response.HandleError(c, err)
		return
	}

	response.Created(c, dto.RegisterResponse{
		UserID:   out.UserID.String(),
		Email:    out.Email,
		Username: out.Username,
		Message:  "Registrasi berhasil. Silakan cek email Anda untuk verifikasi akun.",
	})
}

// VerifyEmail menangani POST /auth/verify-email?token=<token>
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Parameter token diperlukan.", nil)
		return
	}

	if err := h.verifyEmailUC.Execute(c.Request.Context(), token); err != nil {
		h.logError(c, "verify-email", err)
		response.HandleError(c, err)
		return
	}

	response.OK(c, gin.H{"message": "Email berhasil diverifikasi. Silakan login."})
}

// ResendVerification menangani POST /auth/resend-verification
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Format request tidak valid.", nil)
		return
	}

	// Always return generic response (anti-enumeration)
	_ = h.resendVerificationUC.Execute(c.Request.Context(), req.Email)
	response.OK(c, gin.H{"message": "Jika email terdaftar, link verifikasi baru telah dikirim."})
}

// Login menangani POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Format request tidak valid.", nil)
		return
	}

	out, err := h.loginUC.Execute(c.Request.Context(), usecase.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
		IsDev:     h.isDev,
	})
	if err != nil {
		h.logError(c, "login", err)
		response.HandleError(c, err)
		return
	}

	h.setRefreshTokenCookie(c, out.RefreshTokenCookie)

	response.OK(c, dto.LoginResponse{
		AccessToken: out.AccessToken,
		TokenType:   out.TokenType,
		ExpiresIn:   out.ExpiresIn,
	})
}

// Refresh menangani POST /auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	rawRefreshToken, err := c.Cookie(refreshTokenCookieName)
	if err != nil || rawRefreshToken == "" {
		response.Fail(c, http.StatusUnauthorized, "INVALID_TOKEN", "Refresh token tidak ditemukan.", nil)
		return
	}

	out, err := h.refreshTokenUC.Execute(c.Request.Context(), usecase.RefreshInput{
		RawRefreshToken: rawRefreshToken,
		UserAgent:       c.Request.UserAgent(),
		IPAddress:       c.ClientIP(),
		IsDev:           h.isDev,
	})
	if err != nil {
		h.logError(c, "refresh", err)
		response.HandleError(c, err)
		return
	}

	h.setRefreshTokenCookie(c, out.RefreshTokenCookie)

	response.OK(c, dto.LoginResponse{
		AccessToken: out.AccessToken,
		TokenType:   out.TokenType,
		ExpiresIn:   out.ExpiresIn,
	})
}

// Logout menangani POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	accessToken := extractBearerToken(c)
	rawRefreshToken, _ := c.Cookie(refreshTokenCookieName)

	_ = h.logoutUC.Execute(c.Request.Context(), usecase.LogoutInput{
		AccessToken:     accessToken,
		RawRefreshToken: rawRefreshToken,
	})

	// Hapus refresh token cookie
	h.clearRefreshTokenCookie(c)

	response.OK(c, gin.H{"message": "Logout berhasil."})
}

// ForgotPassword menangani POST /auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Format request tidak valid.", nil)
		return
	}

	// Always return generic response (anti-enumeration)
	_ = h.forgotPasswordUC.Execute(c.Request.Context(), req.Email)
	response.OK(c, gin.H{"message": "Jika email terdaftar, instruksi reset password telah dikirim."})
}

// ResetPassword menangani POST /auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Format request tidak valid.", nil)
		return
	}

	if err := h.resetPasswordUC.Execute(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		h.logError(c, "reset-password", err)
		response.HandleError(c, err)
		return
	}

	response.OK(c, gin.H{"message": "Password berhasil direset. Silakan login dengan password baru."})
}

// Health menangani GET /health
func (h *AuthHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	deps := make(map[string]dto.HealthDependency)
	allHealthy := true

	if err := h.healthChecker.PingPostgres(ctx); err != nil {
		deps["postgres"] = dto.HealthDependency{Status: "unhealthy"}
		allHealthy = false
	} else {
		deps["postgres"] = dto.HealthDependency{Status: "ok"}
	}

	if err := h.healthChecker.PingRedis(ctx); err != nil {
		deps["redis"] = dto.HealthDependency{Status: "unhealthy"}
		allHealthy = false
	} else {
		deps["redis"] = dto.HealthDependency{Status: "ok"}
	}

	if err := h.healthChecker.PingRabbitMQ(); err != nil {
		deps["rabbitmq"] = dto.HealthDependency{Status: "unhealthy"}
		allHealthy = false
	} else {
		deps["rabbitmq"] = dto.HealthDependency{Status: "ok"}
	}

	status := "healthy"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"success": allHealthy,
		"data": dto.HealthResponse{
			Status:       status,
			Service:      "auth-service",
			Version:      h.serviceVersion,
			Dependencies: deps,
		},
	})
}

// setRefreshTokenCookie mengatur HttpOnly cookie untuk refresh token.
func (h *AuthHandler) setRefreshTokenCookie(c *gin.Context, params usecase.RefreshTokenCookieParams) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookieName,
		params.RawToken,
		params.MaxAge,
		refreshTokenCookiePath,
		"",
		params.IsSecure,
		true, // httpOnly
	)
}

// clearRefreshTokenCookie menghapus refresh token cookie dengan MaxAge=0.
func (h *AuthHandler) clearRefreshTokenCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshTokenCookieName, "", -1, refreshTokenCookiePath, "", !h.isDev, true)
}

// extractBearerToken mengambil raw token dari Authorization: Bearer <token> header.
func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

// logError mencatat error ke logger dengan context yang diperlukan.
// Tidak pernah log password, token mentah, atau PII lengkap.
func (h *AuthHandler) logError(c *gin.Context, operation string, err error) {
	h.logger.Error().
		Str("operation", operation).
		Str("correlation_id", middleware.GetCorrelationID(c)).
		Str("path", c.Request.URL.Path).
		Err(err).
		Msg("auth operation failed")
}
