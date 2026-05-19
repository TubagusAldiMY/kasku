package usecase_test

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/transaction-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDeleteTransactionUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — SoftDelete berhasil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()
		txID := uuid.New().String()

		txRepo.EXPECT().SoftDelete(gomock.Any(), testTenant, txID, userID).Return(nil)

		uc := usecase.NewDeleteTransactionUseCase(txRepo)
		err := uc.Execute(context.Background(), testTenant, txID, userID)
		require.NoError(t, err)
	})

	t.Run("transaksi tidak ditemukan — ErrTransactionNotFound dipropagasi", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()
		txID := uuid.New().String()

		txRepo.EXPECT().SoftDelete(gomock.Any(), testTenant, txID, userID).Return(domainerrors.ErrTransactionNotFound)

		uc := usecase.NewDeleteTransactionUseCase(txRepo)
		err := uc.Execute(context.Background(), testTenant, txID, userID)
		assert.Error(t, err)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()
		txID := uuid.New().String()

		txRepo.EXPECT().SoftDelete(gomock.Any(), testTenant, txID, userID).Return(errors.New("db error"))

		uc := usecase.NewDeleteTransactionUseCase(txRepo)
		err := uc.Execute(context.Background(), testTenant, txID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gagal hapus transaksi")
	})
}
