package usecase_test

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/finance-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testTenant = "tenant_550e8400_e29b_41d4_a716_446655440000"

func testUserID() string { return uuid.New().String() }

func TestCreateAccountUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — akun berhasil dibuat dengan IDR default", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()

		repo.EXPECT().CountByUserID(gomock.Any(), testTenant, userID).Return(2, nil)
		repo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(nil)

		uc := usecase.NewCreateAccountUseCase(repo)
		input := usecase.CreateAccountInput{
			TenantSchema: testTenant,
			UserID:       userID,
			Name:         "BCA Utama",
			MaxAccounts:  5,
		}
		acc, err := uc.Execute(context.Background(), input)
		require.NoError(t, err)
		assert.Equal(t, "IDR", acc.Currency)
		assert.Equal(t, "BCA Utama", acc.Name)
	})

	t.Run("tier limit -1 (Pro unlimited) — CountByUserID tidak dipanggil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()

		// Tidak ada EXPECT CountByUserID — gomock fail jika dipanggil
		repo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(nil)

		uc := usecase.NewCreateAccountUseCase(repo)
		input := usecase.CreateAccountInput{
			TenantSchema: testTenant,
			UserID:       userID,
			Name:         "Mandiri Pro",
			MaxAccounts:  -1,
		}
		_, err := uc.Execute(context.Background(), input)
		require.NoError(t, err)
	})

	t.Run("tier limit tercapai — ErrAccountLimitReached", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()

		repo.EXPECT().CountByUserID(gomock.Any(), testTenant, userID).Return(3, nil)

		uc := usecase.NewCreateAccountUseCase(repo)
		input := usecase.CreateAccountInput{
			TenantSchema: testTenant,
			UserID:       userID,
			Name:         "Rekening Baru",
			MaxAccounts:  3,
		}
		_, err := uc.Execute(context.Background(), input)
		assert.ErrorIs(t, err, domainerrors.ErrAccountLimitReached)
	})

	t.Run("nama kosong — ErrInvalidInput", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)

		uc := usecase.NewCreateAccountUseCase(repo)
		_, err := uc.Execute(context.Background(), usecase.CreateAccountInput{
			TenantSchema: testTenant,
			UserID:       testUserID(),
			Name:         "",
			MaxAccounts:  5,
		})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})

	t.Run("repo error saat Create — propagate error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()

		repo.EXPECT().CountByUserID(gomock.Any(), testTenant, userID).Return(0, nil)
		repo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(errors.New("db error"))

		uc := usecase.NewCreateAccountUseCase(repo)
		_, err := uc.Execute(context.Background(), usecase.CreateAccountInput{
			TenantSchema: testTenant, UserID: userID, Name: "Test", MaxAccounts: 5,
		})
		assert.Error(t, err)
	})
}
