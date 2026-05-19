package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/transaction-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetTransactionUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("transaksi ditemukan — dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()
		txID := uuid.New().String()

		expected := &entity.Transaction{
			ID: uuid.MustParse(txID), AmountIDR: 75_000,
			TransactionType: entity.TransactionExpense, CreatedAt: time.Now(),
		}
		txRepo.EXPECT().GetByID(gomock.Any(), testTenant, txID, userID).Return(expected, nil)

		uc := usecase.NewGetTransactionUseCase(txRepo)
		tx, err := uc.Execute(context.Background(), testTenant, txID, userID)
		require.NoError(t, err)
		assert.Equal(t, int64(75_000), tx.AmountIDR)
	})

	t.Run("transaksi tidak ada — repo mengembalikan nil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()
		txID := uuid.New().String()

		txRepo.EXPECT().GetByID(gomock.Any(), testTenant, txID, userID).Return(nil, nil)

		uc := usecase.NewGetTransactionUseCase(txRepo)
		tx, err := uc.Execute(context.Background(), testTenant, txID, userID)
		require.NoError(t, err)
		assert.Nil(t, tx)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()
		txID := uuid.New().String()

		txRepo.EXPECT().GetByID(gomock.Any(), testTenant, txID, userID).Return(nil, errors.New("db error"))

		uc := usecase.NewGetTransactionUseCase(txRepo)
		_, err := uc.Execute(context.Background(), testTenant, txID, userID)
		assert.Error(t, err)
	})
}
