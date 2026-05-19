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

func TestGetBalanceHistoryUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — history dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()
		existing := &entity.FinancialAccount{ID: uuid.MustParse(accountID), CreatedAt: time.Now()}
		history := []entity.BalanceHistory{
			{ID: uuid.New(), Amount: 100000, CreatedAt: time.Now()},
		}

		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID, userID).Return(existing, nil)
		repo.EXPECT().GetBalanceHistory(gomock.Any(), testTenant, accountID, 3).Return(history, nil)

		uc := usecase.NewGetBalanceHistoryUseCase(repo)
		result, err := uc.Execute(context.Background(), testTenant, accountID, userID, 3)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("historyMonths negatif — dikonversi ke 0 (unlimited)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()
		existing := &entity.FinancialAccount{ID: uuid.MustParse(accountID), CreatedAt: time.Now()}

		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID, userID).Return(existing, nil)
		repo.EXPECT().GetBalanceHistory(gomock.Any(), testTenant, accountID, 0).Return([]entity.BalanceHistory{}, nil)

		uc := usecase.NewGetBalanceHistoryUseCase(repo)
		_, err := uc.Execute(context.Background(), testTenant, accountID, userID, -1)
		require.NoError(t, err)
	})

	t.Run("akun tidak ditemukan — GetByID error dipropagasi", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()

		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID, userID).Return(nil, errors.New("not found"))

		uc := usecase.NewGetBalanceHistoryUseCase(repo)
		_, err := uc.Execute(context.Background(), testTenant, accountID, userID, 3)
		assert.Error(t, err)
	})
}
