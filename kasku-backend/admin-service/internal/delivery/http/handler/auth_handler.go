package handler

import (
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/dto"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/middleware"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/response"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler menangani login/logout/me untuk admin.
type AuthHandler struct {
	login   usecase.LoginUseCase
	logout  usecase.LogoutUseCase
	current usecase.GetCurrentAdminUseCase
}

// NewAuthHandler membuat instance.
func NewAuthHandler(login usecase.LoginUseCase, logout usecase.LogoutUseCase, current usecase.GetCurrentAdminUseCase) *AuthHandler {
	return &AuthHandler{login: login, logout: logout, current: current}
}

// Login menangani POST /v1/admin/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, domainerrors.ErrValidation.Code, domainerrors.ErrValidation.Message)
		return
	}

	out, err := h.login.Execute(c.Request.Context(), usecase.LoginInput{
		Username: req.Username,
		Password: req.Password,
		IP:       c.ClientIP(),
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.OK(c, dto.LoginResponse{
		AccessToken: out.AccessToken,
		TokenType:   out.TokenType,
		ExpiresIn:   out.ExpiresIn,
		Admin: dto.AdminUserDTO{
			ID:          out.Admin.ID.String(),
			Username:    out.Admin.Username,
			Role:        string(out.Admin.Role),
			IsActive:    out.Admin.IsActive,
			LastLoginAt: out.Admin.LastLoginAt,
			CreatedAt:   out.Admin.CreatedAt,
		},
	})
}

// Logout menangani POST /v1/admin/auth/logout.
// Admin context (ID/JTI/Exp) sudah di-inject AdminAuthMiddleware.
func (h *AuthHandler) Logout(c *gin.Context) {
	adminIDStr := c.GetString(middleware.ContextKeyAdminID)
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		response.Fail(c, 401, domainerrors.ErrUnauthorized.Code, domainerrors.ErrUnauthorized.Message)
		return
	}
	jti := c.GetString(middleware.ContextKeyAdminJTI)
	expVal, _ := c.Get(middleware.ContextKeyAdminExp)
	exp, _ := expVal.(time.Time)

	if err := h.logout.Execute(c.Request.Context(), usecase.LogoutInput{
		AdminID:   adminID,
		JTI:       jti,
		ExpiresAt: exp,
	}); err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "logout berhasil"})
}

// Me menangani GET /v1/admin/auth/me.
func (h *AuthHandler) Me(c *gin.Context) {
	adminIDStr := c.GetString(middleware.ContextKeyAdminID)
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		response.Fail(c, 401, domainerrors.ErrUnauthorized.Code, domainerrors.ErrUnauthorized.Message)
		return
	}
	admin, err := h.current.Execute(c.Request.Context(), adminID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, dto.AdminUserDTO{
		ID:          admin.ID.String(),
		Username:    admin.Username,
		Role:        string(admin.Role),
		IsActive:    admin.IsActive,
		LastLoginAt: admin.LastLoginAt,
		CreatedAt:   admin.CreatedAt,
	})
}
