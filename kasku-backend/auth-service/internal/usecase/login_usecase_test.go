package usecase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
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

// Argon2Config dengan parameter ringan untuk test cepat.
var testArgon2Cfg = usecase.Argon2Config{Time: 1, MemoryKB: 8 * 1024, Threads: 2, KeyLength: 32}

func newTestLoginInput(email, pass string) usecase.LoginInput {
	return usecase.LoginInput{
		Email:     email,
		Password:  pass,
		UserAgent: "test-agent",
		IPAddress: "1.2.3.4",
		IsDev:     true,
	}
}

func TestLoginUseCase_Execute(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	correctPass := "Password123"
	wrongPass := "WrongPass456"

	correctHash, err := usecase.HashPasswordForTest(correctPass, testArgon2Cfg)
	require.NoError(t, err)

	userID := uuid.New()

	t.Run("user not found: dummy verify + ErrInvalidCredentials", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)
		ur.EXPECT().FindByEmail(gomock.Any(), "ghost@example.com").Return(nil, nil)

		uc := usecase.NewLoginUseCase(ur, rr, priv, 15*time.Minute, 24*time.Hour, testArgon2Cfg, 5, 15*time.Minute)
		_, err := uc.Execute(context.Background(), newTestLoginInput("ghost@example.com", wrongPass))
		assert.ErrorIs(t, err, domainerrors.ErrInvalidCredentials)
	})

	t.Run("account locked: ErrAccountLocked", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)
		future := time.Now().UTC().Add(10 * time.Minute)
		user := &entity.User{
			ID:           userID,
			Email:        "locked@example.com",
			PasswordHash: correctHash,
			IsActive:     true,
			LockedUntil:  &future,
		}
		ur.EXPECT().FindByEmail(gomock.Any(), "locked@example.com").Return(user, nil)

		uc := usecase.NewLoginUseCase(ur, rr, priv, 15*time.Minute, 24*time.Hour, testArgon2Cfg, 5, 15*time.Minute)
		_, err := uc.Execute(context.Background(), newTestLoginInput("locked@example.com", correctPass))
		assert.ErrorIs(t, err, domainerrors.ErrAccountLocked)
	})

	t.Run("wrong password: increment counter + ErrInvalidCredentials", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)
		user := &entity.User{
			ID:           userID,
			Email:        "u@example.com",
			PasswordHash: correctHash,
			IsActive:     true,
		}
		ur.EXPECT().FindByEmail(gomock.Any(), "u@example.com").Return(user, nil)
		ur.EXPECT().
			IncrementFailedLoginAndLockIfNeeded(gomock.Any(), userID, int16(5), gomock.Any()).
			Return(nil)

		uc := usecase.NewLoginUseCase(ur, rr, priv, 15*time.Minute, 24*time.Hour, testArgon2Cfg, 5, 15*time.Minute)
		_, err := uc.Execute(context.Background(), newTestLoginInput("u@example.com", wrongPass))
		assert.ErrorIs(t, err, domainerrors.ErrInvalidCredentials)
	})

	t.Run("password correct but inactive: ErrAccountNotVerified", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)
		user := &entity.User{
			ID:           userID,
			Email:        "u@example.com",
			PasswordHash: correctHash,
			IsActive:     false,
		}
		ur.EXPECT().FindByEmail(gomock.Any(), "u@example.com").Return(user, nil)
		// reset counter meski belum aktif
		ur.EXPECT().UpdateLoginSuccess(gomock.Any(), userID).Return(nil)

		uc := usecase.NewLoginUseCase(ur, rr, priv, 15*time.Minute, 24*time.Hour, testArgon2Cfg, 5, 15*time.Minute)
		_, err := uc.Execute(context.Background(), newTestLoginInput("u@example.com", correctPass))
		assert.ErrorIs(t, err, domainerrors.ErrAccountNotVerified)
	})

	t.Run("happy: issue JWT + refresh token", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)
		user := &entity.User{
			ID:           userID,
			Email:        "u@example.com",
			PasswordHash: correctHash,
			IsActive:     true,
		}
		ur.EXPECT().FindByEmail(gomock.Any(), "u@example.com").Return(user, nil)
		ur.EXPECT().UpdateLoginSuccess(gomock.Any(), userID).Return(nil)
		rr.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		uc := usecase.NewLoginUseCase(ur, rr, priv, 15*time.Minute, 24*time.Hour, testArgon2Cfg, 5, 15*time.Minute)
		out, err := uc.Execute(context.Background(), newTestLoginInput("u@example.com", correctPass))
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.NotEmpty(t, out.AccessToken)
		assert.NotEmpty(t, out.RefreshTokenCookie.RawToken)
		assert.Equal(t, "Bearer", out.TokenType)
		assert.Equal(t, int64(900), out.ExpiresIn) // 15min
	})
}

// TestLoginUseCase_TimingAttack memverifikasi response latency untuk user-not-found
// vs wrong-password tidak berbeda signifikan. Penyerang yang mengukur waktu respons
// tidak boleh bisa enumerasi email valid.
//
// Note: ini bukan jaminan absolut karena Go GC + jitter, tapi mendeteksi regresi
// kasar (mis. kalau seseorang menghilangkan runDummyArgon2Verify).
func TestLoginUseCase_TimingAttack_ConstantTimeCheck(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	correctHash, err := usecase.HashPasswordForTest("CorrectPass1", testArgon2Cfg)
	require.NoError(t, err)
	userID := uuid.New()

	const iterations = 3 // hashing argon2 is slow; minimal iterations for sanity check
	measure := func(setup func(*mocks.MockUserRepository)) time.Duration {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockRefreshTokenRepository(ctrl)
		setup(ur)

		uc := usecase.NewLoginUseCase(ur, rr, priv, 15*time.Minute, 24*time.Hour, testArgon2Cfg, 5, 15*time.Minute)

		total := time.Duration(0)
		for range iterations {
			start := time.Now()
			_, _ = uc.Execute(context.Background(), newTestLoginInput("x@y.com", "WrongPass1"))
			total += time.Since(start)
		}
		return total / iterations
	}

	tNotFound := measure(func(ur *mocks.MockUserRepository) {
		ur.EXPECT().FindByEmail(gomock.Any(), "x@y.com").Return(nil, nil).Times(iterations)
	})
	tWrongPass := measure(func(ur *mocks.MockUserRepository) {
		user := &entity.User{ID: userID, Email: "x@y.com", PasswordHash: correctHash, IsActive: true}
		ur.EXPECT().FindByEmail(gomock.Any(), "x@y.com").Return(user, nil).Times(iterations)
		ur.EXPECT().
			IncrementFailedLoginAndLockIfNeeded(gomock.Any(), userID, int16(5), gomock.Any()).
			Return(nil).Times(iterations)
	})

	// Tolerance: 3x deviation acceptable mengingat Go GC + Argon2 jitter.
	// Tujuan utama: detect regression kalau dummy verify dihapus (akan jadi
	// 10-100x lebih cepat). Test ini sengaja loose untuk hindari flakiness.
	ratio := float64(tNotFound) / float64(tWrongPass)
	assert.GreaterOrEqualf(t, ratio, 0.33,
		"not-found path terlalu cepat — dummy verify mungkin dihapus: not-found=%v, wrong-pass=%v", tNotFound, tWrongPass)
	assert.LessOrEqualf(t, ratio, 3.0,
		"not-found path terlalu lambat — investigate: not-found=%v, wrong-pass=%v", tNotFound, tWrongPass)
}
