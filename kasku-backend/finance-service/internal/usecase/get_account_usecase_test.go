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

func TestGetAccountUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("akun ditemukan — dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()

		expected := &entity.FinancialAccount{ID: uuid.MustParse(accountID), Name: "Savings", CreatedAt: time.Now()}
		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID, userID).Return(expected, nil)

		uc := usecase.NewGetAccountUseCase(repo)
		acc, err := uc.Execute(context.Background(), testTenant, accountID, userID)
		require.NoError(t, err)
		assert.Equal(t, "Savings", acc.Name)
	})

	t.Run("akun tidak ditemukan — repo mengembalikan nil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()

		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID, userID).Return(nil, nil)

		uc := usecase.NewGetAccountUseCase(repo)
		acc, err := uc.Execute(context.Background(), testTenant, accountID, userID)
		require.NoError(t, err)
		assert.Nil(t, acc)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()

		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID, userID).Return(nil, errors.New("db error"))

		uc := usecase.NewGetAccountUseCase(repo)
		_, err := uc.Execute(context.Background(), testTenant, accountID, userID)
		assert.Error(t, err)
	})
}
