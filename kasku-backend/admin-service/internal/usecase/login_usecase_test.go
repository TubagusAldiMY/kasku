package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testPassword = "SuperSecret123!"

// testAdminUser mengembalikan AdminUser dengan password yang sudah di-hash menggunakan
// Argon2Config fast (parameter test). Hash ini dibuat setiap kali agar selalu valid.
func testAdminUser(t *testing.T) *entity.AdminUser {
	t.Helper()
	hash, err := usecase.HashPassword(testPassword, testArgon2())
	require.NoError(t, err)
	return &entity.AdminUser{
		ID:           uuid.New(),
		Username:     "admin",
		PasswordHash: hash,
		Role:         entity.AdminRoleSuperAdmin,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func TestLoginUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — token dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAdminUserRepository(ctrl)
		auditLogger, mockAudit := testAuditLogger(ctrl)
		admin := testAdminUser(t)

		repo.EXPECT().FindByUsername(gomock.Any(), "admin").Return(admin, nil)
		repo.EXPECT().UpdateLastLogin(gomock.Any(), admin.ID, gomock.Any()).Return(nil)
		mockAudit.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		uc := usecase.NewLoginUseCase(repo, testSigner(), testArgon2(), auditLogger)
		out, err := uc.Execute(context.Background(), usecase.LoginInput{
			Username: "admin", Password: testPassword, IP: "127.0.0.1",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, out.AccessToken)
		assert.Equal(t, "Bearer", out.TokenType)
	})

	t.Run("user tidak ditemukan — ErrInvalidCredentials (timing-safe)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAdminUserRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)

		repo.EXPECT().FindByUsername(gomock.Any(), "notexist").Return(nil, nil)
		// runDummyVerify dipanggil internal — tidak ada repo call lain

		uc := usecase.NewLoginUseCase(repo, testSigner(), testArgon2(), auditLogger)
		_, err := uc.Execute(context.Background(), usecase.LoginInput{
			Username: "notexist", Password: testPassword,
		})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidCredentials)
	})

	t.Run("password salah — ErrInvalidCredentials", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAdminUserRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)
		admin := testAdminUser(t)

		repo.EXPECT().FindByUsername(gomock.Any(), admin.Username).Return(admin, nil)

		uc := usecase.NewLoginUseCase(repo, testSigner(), testArgon2(), auditLogger)
		_, err := uc.Execute(context.Background(), usecase.LoginInput{
			Username: admin.Username, Password: "WrongPassword!",
		})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidCredentials)
	})

	t.Run("admin tidak aktif — ErrAdminInactive", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAdminUserRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)
		admin := testAdminUser(t)
		admin.IsActive = false

		repo.EXPECT().FindByUsername(gomock.Any(), admin.Username).Return(admin, nil)

		uc := usecase.NewLoginUseCase(repo, testSigner(), testArgon2(), auditLogger)
		_, err := uc.Execute(context.Background(), usecase.LoginInput{
			Username: admin.Username, Password: testPassword,
		})
		assert.ErrorIs(t, err, domainerrors.ErrAdminInactive)
	})

	t.Run("UpdateLastLogin gagal — login tetap berhasil (non-fatal)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAdminUserRepository(ctrl)
		auditLogger, mockAudit := testAuditLogger(ctrl)
		admin := testAdminUser(t)

		repo.EXPECT().FindByUsername(gomock.Any(), admin.Username).Return(admin, nil)
		repo.EXPECT().UpdateLastLogin(gomock.Any(), admin.ID, gomock.Any()).Return(errors.New("db timeout"))
		mockAudit.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		uc := usecase.NewLoginUseCase(repo, testSigner(), testArgon2(), auditLogger)
		out, err := uc.Execute(context.Background(), usecase.LoginInput{
			Username: admin.Username, Password: testPassword, IP: "10.0.0.1",
		})
		// Login harus tetap berhasil meskipun last_login update gagal
		require.NoError(t, err)
		assert.NotEmpty(t, out.AccessToken)
	})

	t.Run("repo error saat FindByUsername — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockAdminUserRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)

		repo.EXPECT().FindByUsername(gomock.Any(), "admin").Return(nil, errors.New("db error"))

		uc := usecase.NewLoginUseCase(repo, testSigner(), testArgon2(), auditLogger)
		_, err := uc.Execute(context.Background(), usecase.LoginInput{
			Username: "admin", Password: testPassword,
		})
		assert.Error(t, err)
	})
}
