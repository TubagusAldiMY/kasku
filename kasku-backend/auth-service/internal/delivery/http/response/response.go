package response

import (
	"errors"
	"net/http"

	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/gin-gonic/gin"
)

// Response adalah struktur standar semua response JSON KasKu.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
}

// ErrorBody berisi detail error yang dikembalikan ke client.
type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// OK mengembalikan response sukses 200.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

// Created mengembalikan response sukses 201.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Success: true, Data: data})
}

// Fail mengembalikan response error dengan status code yang diberikan.
func Fail(c *gin.Context, statusCode int, code, message string, details interface{}) {
	c.JSON(statusCode, Response{
		Success: false,
		Error:   &ErrorBody{Code: code, Message: message, Details: details},
	})
}

// HandleError memetakan error ke HTTP response yang sesuai.
// Domain errors mendapat HTTP status yang spesifik, error lain mendapat 500.
func HandleError(c *gin.Context, err error) {
	var domainErr *domainerrors.DomainError
	if errors.As(err, &domainErr) {
		statusCode := domainErrorToStatus(domainErr.Code)
		Fail(c, statusCode, domainErr.Code, domainErr.Message, nil)
		return
	}
	Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan internal.", nil)
}

// domainErrorToStatus memetakan error code ke HTTP status code.
func domainErrorToStatus(code string) int {
	switch code {
	case "INVALID_CREDENTIALS", "INVALID_TOKEN", "TOKEN_EXPIRED":
		return http.StatusUnauthorized
	case "ACCOUNT_LOCKED", "ACCOUNT_NOT_VERIFIED":
		return http.StatusForbidden
	case "TOKEN_REUSE_DETECTED":
		return http.StatusUnauthorized
	case "EMAIL_ALREADY_EXISTS", "USERNAME_ALREADY_EXISTS", "EMAIL_ALREADY_VERIFIED":
		return http.StatusConflict
	case "PASSWORD_TOO_SHORT", "PASSWORD_TOO_WEAK", "VALIDATION_ERROR":
		return http.StatusBadRequest
	case "USER_NOT_FOUND":
		return http.StatusNotFound
	case "SERVICE_UNAVAILABLE":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
