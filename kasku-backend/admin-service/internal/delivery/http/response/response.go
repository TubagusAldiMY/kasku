package response

import (
	"errors"
	"net/http"

	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/gin-gonic/gin"
)

// Envelope sukses: {"success": true, "data": ..., "meta": ... (opsional)}.
// Envelope gagal: {"success": false, "error": {"code": "...", "message": "..."}}.

// OK menulis 200 dengan data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// OKMeta menulis 200 dengan data + meta (pagination dll).
func OKMeta(c *gin.Context, data any, meta any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "meta": meta})
}

// Created menulis 201.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": data})
}

// HandleError memetakan error ke envelope gagal + status HTTP yang tepat.
func HandleError(c *gin.Context, err error) {
	var derr *domainerrors.DomainError
	if errors.As(err, &derr) {
		status := codeToStatus(derr.Code)
		Fail(c, status, derr.Code, derr.Message)
		return
	}
	// Generic internal error — pesan disanitasi.
	Fail(c, http.StatusInternalServerError, domainerrors.ErrInternal.Code, domainerrors.ErrInternal.Message)
}

// Fail menulis envelope error eksplisit.
func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error":   gin.H{"code": code, "message": message},
	})
}

func codeToStatus(code string) int {
	switch code {
	case domainerrors.ErrInvalidCredentials.Code,
		domainerrors.ErrInvalidToken.Code,
		domainerrors.ErrTokenRevoked.Code,
		domainerrors.ErrUnauthorized.Code:
		return http.StatusUnauthorized
	case domainerrors.ErrAdminInactive.Code,
		domainerrors.ErrForbidden.Code:
		return http.StatusForbidden
	case domainerrors.ErrUserNotFound.Code,
		domainerrors.ErrAdminNotFound.Code,
		domainerrors.ErrSubscriptionNotFound.Code,
		domainerrors.ErrPlanNotFound.Code:
		return http.StatusNotFound
	case domainerrors.ErrValidation.Code:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
