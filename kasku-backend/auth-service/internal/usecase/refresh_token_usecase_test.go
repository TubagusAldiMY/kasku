package usecase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRefreshTokenUseCase_Execute(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	accessTTL := 15 * time.Minute
	refreshTTL := 24 * time.Hour

	rawToken := "raw-refresh-abc"
	tokenHash := usecase.HashTokenForTest(rawToken)
	tokenID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	activeToken := &entity.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(1 * time.Hour),
		IsRevoked: false,
		CreatedAt: now,
	}

	t.Run("empty token → ErrInvalidToken", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)

		uc := usecase.NewRefreshTokenUseCase(ur, rr, priv, accessTTL, refreshTTL)
		_, err := uc.Execute(context.Background(), usecase.RefreshInput{RawRefreshToken: ""})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidToken)
	})

	t.Run("token not found → ErrInvalidToken", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)
		rr.EXPECT().FindByTokenHash(gomock.Any(), tokenHash).Return(nil, nil)

		uc := usecase.NewRefreshTokenUseCase(ur, rr, priv, accessTTL, refreshTTL)
		_, err := uc.Execute(context.Background(), usecase.RefreshInput{RawRefreshToken: rawToken})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidToken)
	})

	t.Run("REUSE detection: revoked token → revoke all + ErrTokenReuseDetected", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)

		revoked := *activeToken
		revoked.IsRevoked = true
		rr.EXPECT().FindByTokenHash(gomock.Any(), tokenHash).Return(&revoked, nil)
		// critical: revoke ALL active tokens of this user (force re-login everywhere)
		rr.EXPECT().RevokeAllActiveByUserID(gomock.Any(), userID).Return(nil)

		uc := usecase.NewRefreshTokenUseCase(ur, rr, priv, accessTTL, refreshTTL)
		_, err := uc.Execute(context.Background(), usecase.RefreshInput{RawRefreshToken: rawToken})
		assert.ErrorIs(t, err, domainerrors.ErrTokenReuseDetected)
	})

	t.Run("expired token → ErrInvalidToken", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)

		expired := *activeToken
		expired.ExpiresAt = now.Add(-1 * time.Hour)
		rr.EXPECT().FindByTokenHash(gomock.Any(), tokenHash).Return(&expired, nil)

		uc := usecase.NewRefreshTokenUseCase(ur, rr, priv, accessTTL, refreshTTL)
		_, err := uc.Execute(context.Background(), usecase.RefreshInput{RawRefreshToken: rawToken})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidToken)
	})

	t.Run("user not found after revoke → ErrInvalidToken", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)

		rr.EXPECT().FindByTokenHash(gomock.Any(), tokenHash).Return(activeToken, nil)
		rr.EXPECT().RevokeByID(gomock.Any(), tokenID).Return(nil)
		ur.EXPECT().FindByID(gomock.Any(), userID).Return(nil, nil)

		uc := usecase.NewRefreshTokenUseCase(ur, rr, priv, accessTTL, refreshTTL)
		_, err := uc.Execute(context.Background(), usecase.RefreshInput{RawRefreshToken: rawToken})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidToken)
	})

	t.Run("happy: rotate token, issue new pair", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)

		user := &entity.User{
			ID:            userID,
			Email:         "u@example.com",
			IsActive:      true,
			EmailVerified: true,
		}

		rr.EXPECT().FindByTokenHash(gomock.Any(), tokenHash).Return(activeToken, nil)
		rr.EXPECT().RevokeByID(gomock.Any(), tokenID).Return(nil)
		ur.EXPECT().FindByID(gomock.Any(), userID).Return(user, nil)
		rr.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		uc := usecase.NewRefreshTokenUseCase(ur, rr, priv, accessTTL, refreshTTL)
		out, err := uc.Execute(context.Background(), usecase.RefreshInput{
			RawRefreshToken: rawToken,
			UserAgent:       "test-agent",
			IPAddress:       "1.2.3.4",
		})
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.NotEmpty(t, out.AccessToken)
		assert.NotEmpty(t, out.RefreshTokenCookie.RawToken)
		assert.Equal(t, int64(accessTTL.Seconds()), out.ExpiresIn)
	})

	t.Run("lookup infra error → wrapped", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)
		rr.EXPECT().FindByTokenHash(gomock.Any(), tokenHash).Return(nil, errors.New("db down"))

		uc := usecase.NewRefreshTokenUseCase(ur, rr, priv, accessTTL, refreshTTL)
		_, err := uc.Execute(context.Background(), usecase.RefreshInput{RawRefreshToken: rawToken})
		assert.Error(t, err)
	})
}
