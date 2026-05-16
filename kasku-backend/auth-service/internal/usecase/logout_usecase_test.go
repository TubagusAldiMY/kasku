package usecase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/mocks"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// genTestRSAKeys membuat keypair RSA 2048 untuk testing (cepat).
// Reused di banyak test file via package-level fixture.
func genTestRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return priv, &priv.PublicKey
}

// signTestJWT membuat JWT dengan claims tertentu, ditandatangani RS256.
func signTestJWT(t *testing.T, priv *rsa.PrivateKey, claims usecase.JWTClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(priv)
	require.NoError(t, err)
	return signed
}

func TestLogoutUseCase_Execute(t *testing.T) {
	t.Parallel()

	priv, pub := genTestRSAKeys(t)
	now := time.Now().UTC()
	validJTI := uuid.NewString()

	validToken := signTestJWT(t, priv, usecase.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.NewString(),
			ID:        validJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
		Email:            "u@example.com",
		TenantSchema:     "tenant_x",
		SubscriptionTier: "FREE",
	})

	rawRefresh := "raw-refresh-token-abc"
	refreshHash := usecase.HashTokenForTest(rawRefresh)
	refreshID := uuid.New()

	tests := []struct {
		name      string
		input     usecase.LogoutInput
		setupMock func(refreshRepo *mocks.MockRefreshTokenRepository, bl *mocks.MockTokenBlacklister)
	}{
		{
			name: "happy: blacklist JTI + revoke refresh",
			input: usecase.LogoutInput{
				AccessToken:     validToken,
				RawRefreshToken: rawRefresh,
			},
			setupMock: func(refreshRepo *mocks.MockRefreshTokenRepository, bl *mocks.MockTokenBlacklister) {
				bl.EXPECT().
					BlacklistJTI(gomock.Any(), validJTI, gomock.Any()).
					Return(nil)
				refreshRepo.EXPECT().
					FindByTokenHash(gomock.Any(), refreshHash).
					Return(&entity.RefreshToken{ID: refreshID, IsRevoked: false}, nil)
				refreshRepo.EXPECT().
					RevokeByID(gomock.Any(), refreshID).
					Return(nil)
			},
		},
		{
			name: "idempotent: refresh already revoked → no RevokeByID call",
			input: usecase.LogoutInput{
				AccessToken:     validToken,
				RawRefreshToken: rawRefresh,
			},
			setupMock: func(refreshRepo *mocks.MockRefreshTokenRepository, bl *mocks.MockTokenBlacklister) {
				bl.EXPECT().BlacklistJTI(gomock.Any(), validJTI, gomock.Any()).Return(nil)
				refreshRepo.EXPECT().
					FindByTokenHash(gomock.Any(), refreshHash).
					Return(&entity.RefreshToken{ID: refreshID, IsRevoked: true}, nil)
				// no RevokeByID — already revoked
			},
		},
		{
			name: "empty access token: skip blacklist, still revoke refresh",
			input: usecase.LogoutInput{
				AccessToken:     "",
				RawRefreshToken: rawRefresh,
			},
			setupMock: func(refreshRepo *mocks.MockRefreshTokenRepository, bl *mocks.MockTokenBlacklister) {
				refreshRepo.EXPECT().
					FindByTokenHash(gomock.Any(), refreshHash).
					Return(&entity.RefreshToken{ID: refreshID, IsRevoked: false}, nil)
				refreshRepo.EXPECT().RevokeByID(gomock.Any(), refreshID).Return(nil)
			},
		},
		{
			name: "empty refresh: skip refresh lookup, still blacklist JTI",
			input: usecase.LogoutInput{
				AccessToken:     validToken,
				RawRefreshToken: "",
			},
			setupMock: func(refreshRepo *mocks.MockRefreshTokenRepository, bl *mocks.MockTokenBlacklister) {
				bl.EXPECT().BlacklistJTI(gomock.Any(), validJTI, gomock.Any()).Return(nil)
			},
		},
		{
			name: "invalid access token: blacklist skipped silently",
			input: usecase.LogoutInput{
				AccessToken:     "garbage.token.here",
				RawRefreshToken: "",
			},
			setupMock: func(_ *mocks.MockRefreshTokenRepository, _ *mocks.MockTokenBlacklister) {
				// no blacklist call — parse fails silently
			},
		},
		{
			name: "blacklist infra error: swallowed, no panic",
			input: usecase.LogoutInput{
				AccessToken:     validToken,
				RawRefreshToken: rawRefresh,
			},
			setupMock: func(refreshRepo *mocks.MockRefreshTokenRepository, bl *mocks.MockTokenBlacklister) {
				bl.EXPECT().BlacklistJTI(gomock.Any(), validJTI, gomock.Any()).Return(errors.New("redis down"))
				refreshRepo.EXPECT().
					FindByTokenHash(gomock.Any(), refreshHash).
					Return(&entity.RefreshToken{ID: refreshID, IsRevoked: false}, nil)
				refreshRepo.EXPECT().RevokeByID(gomock.Any(), refreshID).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			refreshRepo := mocks.NewMockRefreshTokenRepository(ctrl)
			bl := mocks.NewMockTokenBlacklister(ctrl)
			tt.setupMock(refreshRepo, bl)

			uc := usecase.NewLogoutUseCase(refreshRepo, pub, bl)
			err := uc.Execute(context.Background(), tt.input)
			assert.NoError(t, err)
		})
	}
}
