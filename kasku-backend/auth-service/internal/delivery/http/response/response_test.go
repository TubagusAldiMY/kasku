package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/response"
	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestOK_Created_Fail(t *testing.T) {
	t.Parallel()

	t.Run("OK 200 + body", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		response.OK(c, gin.H{"x": 1})
		require.Equal(t, http.StatusOK, w.Code)

		var resp response.Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.True(t, resp.Success)
	})

	t.Run("Created 201", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		response.Created(c, gin.H{"id": "abc"})
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Fail with code+message+details", func(t *testing.T) {
		t.Parallel()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		response.Fail(c, http.StatusBadRequest, "BAD", "bad input", gin.H{"f": "email"})
		require.Equal(t, http.StatusBadRequest, w.Code)

		var resp response.Response
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.False(t, resp.Success)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "BAD", resp.Error.Code)
	})
}

// TestHandleError memetakan setiap domain error code ke HTTP status — exercise
// semua branch domainErrorToStatus untuk coverage 100%.
func TestHandleError_DomainMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"InvalidCredentials → 401", domainerrors.ErrInvalidCredentials, http.StatusUnauthorized},
		{"InvalidToken → 401", domainerrors.ErrInvalidToken, http.StatusUnauthorized},
		{"AccountLocked → 403", domainerrors.ErrAccountLocked, http.StatusForbidden},
		{"AccountNotVerified → 403", domainerrors.ErrAccountNotVerified, http.StatusForbidden},
		{"TokenReuseDetected → 401", domainerrors.ErrTokenReuseDetected, http.StatusUnauthorized},
		{"EmailAlreadyExists → 409", domainerrors.ErrEmailAlreadyExists, http.StatusConflict},
		{"UsernameAlreadyExists → 409", domainerrors.ErrUsernameAlreadyExists, http.StatusConflict},
		{"EmailAlreadyVerified → 409", domainerrors.ErrEmailAlreadyVerified, http.StatusConflict},
		{"PasswordTooShort → 400", domainerrors.ErrPasswordTooShort, http.StatusBadRequest},
		{"PasswordTooWeak → 400", domainerrors.ErrPasswordTooWeak, http.StatusBadRequest},
		{"Validation → 400", domainerrors.ErrValidation, http.StatusBadRequest},
		{"UserNotFound → 404", domainerrors.ErrUserNotFound, http.StatusNotFound},
		{"ServiceUnavailable → 503", domainerrors.ErrServiceUnavailable, http.StatusServiceUnavailable},
		{"Internal → 500", domainerrors.ErrInternal, http.StatusInternalServerError},
		{"non-domain → 500", errors.New("plain error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			response.HandleError(c, tt.err)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
