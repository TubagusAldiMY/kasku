package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/finance-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListAccountsUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — daftar akun dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()

		expected := []entity.FinancialAccount{
			{ID: uuid.New(), Name: "BCA", CreatedAt: time.Now()},
			{ID: uuid.New(), Name: "Mandiri", CreatedAt: time.Now()},
		}
		repo.EXPECT().List(gomock.Any(), testTenant, userID).Return(expected, nil)

		uc := usecase.NewListAccountsUseCase(repo)
		accounts, err := uc.Execute(context.Background(), testTenant, userID)
		require.NoError(t, err)
		assert.Len(t, accounts, 2)
	})

	t.Run("daftar kosong — dikembalikan tanpa error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()

		repo.EXPECT().List(gomock.Any(), testTenant, userID).Return([]entity.FinancialAccount{}, nil)

		uc := usecase.NewListAccountsUseCase(repo)
		accounts, err := uc.Execute(context.Background(), testTenant, userID)
		require.NoError(t, err)
		assert.Empty(t, accounts)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()

		repo.EXPECT().List(gomock.Any(), testTenant, userID).Return(nil, errors.New("db error"))

		uc := usecase.NewListAccountsUseCase(repo)
		_, err := uc.Execute(context.Background(), testTenant, userID)
		assert.Error(t, err)
	})
}
