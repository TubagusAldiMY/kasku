package usecase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/mocks"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func makeJWT(t *testing.T, priv *rsa.PrivateKey, claims usecase.JWTClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(priv)
	require.NoError(t, err)
	return s
}

func TestValidateAccessTokenUseCase_Execute(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pub := &priv.PublicKey

	// other key for wrong-signature test
	otherPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	now := time.Now().UTC()
	validJTI := uuid.NewString()
	validUserID := uuid.NewString()

	validClaims := usecase.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   validUserID,
			ID:        validJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
		Email:            "u@example.com",
		TenantSchema:     "tenant_x",
		SubscriptionTier: "FREE",
	}

	expiredClaims := validClaims
	expiredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(-1 * time.Minute))

	tests := []struct {
		name      string
		token     string
		setupMock func(bl *mocks.MockTokenBlacklistChecker)
		wantErr   error
	}{
		{
			name:  "happy: valid + not blacklisted",
			token: makeJWT(t, priv, validClaims),
			setupMock: func(bl *mocks.MockTokenBlacklistChecker) {
				bl.EXPECT().IsJTIBlacklisted(gomock.Any(), validJTI).Return(false, nil)
			},
			wantErr: nil,
		},
		{
			name:      "empty token → ErrInvalidToken",
			token:     "",
			setupMock: func(_ *mocks.MockTokenBlacklistChecker) {},
			wantErr:   domainerrors.ErrInvalidToken,
		},
		{
			name:      "malformed token → ErrInvalidToken",
			token:     "not.a.jwt",
			setupMock: func(_ *mocks.MockTokenBlacklistChecker) {},
			wantErr:   domainerrors.ErrInvalidToken,
		},
		{
			name:      "expired token → ErrInvalidToken",
			token:     makeJWT(t, priv, expiredClaims),
			setupMock: func(_ *mocks.MockTokenBlacklistChecker) {},
			wantErr:   domainerrors.ErrInvalidToken,
		},
		{
			name:      "wrong signature → ErrInvalidToken",
			token:     makeJWT(t, otherPriv, validClaims),
			setupMock: func(_ *mocks.MockTokenBlacklistChecker) {},
			wantErr:   domainerrors.ErrInvalidToken,
		},
		{
			name:  "blacklisted JTI → ErrInvalidToken",
			token: makeJWT(t, priv, validClaims),
			setupMock: func(bl *mocks.MockTokenBlacklistChecker) {
				bl.EXPECT().IsJTIBlacklisted(gomock.Any(), validJTI).Return(true, nil)
			},
			wantErr: domainerrors.ErrInvalidToken,
		},
		{
			name:  "blacklist infra error → wrapped error",
			token: makeJWT(t, priv, validClaims),
			setupMock: func(bl *mocks.MockTokenBlacklistChecker) {
				bl.EXPECT().IsJTIBlacklisted(gomock.Any(), validJTI).Return(false, errors.New("redis down"))
			},
			// wantErr asserted as "not nil" below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			bl := mocks.NewMockTokenBlacklistChecker(ctrl)
			tt.setupMock(bl)

			uc := usecase.NewValidateAccessTokenUseCase(pub, bl)
			claims, err := uc.Execute(context.Background(), tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, claims)
				return
			}
			if tt.name == "blacklist infra error → wrapped error" {
				assert.Error(t, err)
				assert.Nil(t, claims)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, claims)
			assert.Equal(t, validJTI, claims.ID)
			assert.Equal(t, validUserID, claims.Subject)
		})
	}
}
