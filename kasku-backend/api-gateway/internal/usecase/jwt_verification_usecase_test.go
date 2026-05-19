package usecase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/api-gateway/internal/usecase"
	"github.com/TubagusAldiMY/kasku/api-gateway/tests/mocks"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// buildToken membantu membuat JWT RS256 yang valid (atau manipulasi tertentu) untuk test.
func buildToken(t *testing.T, priv *rsa.PrivateKey, claims jwt.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(priv)
	require.NoError(t, err)
	return signed
}

func validClaims(jti string) usecase.KasKuClaims {
	return usecase.KasKuClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(15 * time.Minute)),
		},
		Email:            "u***@example.com",
		TenantSchema:     "tenant_550e8400_e29b_41d4_a716_446655440000",
		SubscriptionTier: "FREE",
	}
}

func TestJWTVerificationUseCase_Verify(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pub := &priv.PublicKey

	// Separate key pair untuk uji algorithm confusion
	wrongPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("valid token — returns ParsedToken tanpa error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		jti := uuid.New().String()
		token := buildToken(t, priv, validClaims(jti))

		bl := mocks.NewMockBlacklistChecker(ctrl)
		bl.EXPECT().IsBlacklisted(gomock.Any(), jti).Return(false, nil)

		uc := usecase.NewJWTVerificationUseCase(pub, bl)
		parsed, err := uc.Verify(context.Background(), token)

		require.NoError(t, err)
		assert.Equal(t, jti, parsed.JTI)
		assert.Equal(t, "tenant_550e8400_e29b_41d4_a716_446655440000", parsed.TenantSchema)
	})

	t.Run("expired token — error tanpa blacklist check", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		jti := uuid.New().String()

		expired := usecase.KasKuClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   uuid.New().String(),
				ID:        jti,
				IssuedAt:  jwt.NewNumericDate(time.Now().UTC().Add(-2 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(-1 * time.Hour)),
			},
			Email: "u***@example.com",
		}
		token := buildToken(t, priv, expired)

		bl := mocks.NewMockBlacklistChecker(ctrl)
		// Blacklist tidak boleh dipanggil untuk token expired

		uc := usecase.NewJWTVerificationUseCase(pub, bl)
		_, err := uc.Verify(context.Background(), token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tidak valid")
	})

	t.Run("token ditandatangani dengan wrong key — error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		jti := uuid.New().String()
		// Tanda tangani dengan wrongPriv, tapi verifikasi dengan pub (dari priv)
		token := buildToken(t, wrongPriv, validClaims(jti))

		bl := mocks.NewMockBlacklistChecker(ctrl)
		// Blacklist tidak boleh dipanggil

		uc := usecase.NewJWTVerificationUseCase(pub, bl)
		_, err := uc.Verify(context.Background(), token)
		assert.Error(t, err)
	})

	t.Run("token JTI ada di blacklist — error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		jti := uuid.New().String()
		token := buildToken(t, priv, validClaims(jti))

		bl := mocks.NewMockBlacklistChecker(ctrl)
		bl.EXPECT().IsBlacklisted(gomock.Any(), jti).Return(true, nil)

		uc := usecase.NewJWTVerificationUseCase(pub, bl)
		_, err := uc.Verify(context.Background(), token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "direvoke")
	})

	t.Run("token tanpa JTI — error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		noJTI := usecase.KasKuClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   uuid.New().String(),
				ID:        "", // JTI kosong
				IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
				ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(15 * time.Minute)),
			},
			Email: "u***@example.com",
		}
		token := buildToken(t, priv, noJTI)

		bl := mocks.NewMockBlacklistChecker(ctrl)
		// IsBlacklisted tidak boleh dipanggil

		uc := usecase.NewJWTVerificationUseCase(pub, bl)
		_, err := uc.Verify(context.Background(), token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JTI")
	})

	t.Run("malformed token string — error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		bl := mocks.NewMockBlacklistChecker(ctrl)

		uc := usecase.NewJWTVerificationUseCase(pub, bl)
		_, err := uc.Verify(context.Background(), "not.a.jwt")
		assert.Error(t, err)
	})
}
