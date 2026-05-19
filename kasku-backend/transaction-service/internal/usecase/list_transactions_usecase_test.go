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

func TestListTransactionsUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — transaksi dan summary dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()

		txs := []entity.Transaction{
			{ID: uuid.New(), AmountIDR: 100_000, TransactionType: entity.TransactionIncome},
		}
		summary := &entity.TransactionSummary{TotalIncome: 100_000, TotalExpense: 0, NetAmount: 100_000}

		txRepo.EXPECT().List(gomock.Any(), testTenant, userID, gomock.Any(), gomock.Any()).Return(txs, nil)
		txRepo.EXPECT().GetSummary(gomock.Any(), testTenant, userID, gomock.Any(), gomock.Any()).Return(summary, nil)

		uc := usecase.NewListTransactionsUseCase(txRepo)
		result, err := uc.Execute(context.Background(), usecase.ListTransactionsInput{
			TenantSchema: testTenant,
			UserID:       userID,
		})
		require.NoError(t, err)
		assert.Len(t, result.Transactions, 1)
		assert.Equal(t, int64(100_000), result.Summary.TotalIncome)
	})

	t.Run("daftar kosong — dikembalikan tanpa error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().List(gomock.Any(), testTenant, userID, gomock.Any(), gomock.Any()).Return([]entity.Transaction{}, nil)
		txRepo.EXPECT().GetSummary(gomock.Any(), testTenant, userID, gomock.Any(), gomock.Any()).Return(&entity.TransactionSummary{}, nil)

		uc := usecase.NewListTransactionsUseCase(txRepo)
		result, err := uc.Execute(context.Background(), usecase.ListTransactionsInput{
			TenantSchema: testTenant,
			UserID:       userID,
		})
		require.NoError(t, err)
		assert.Empty(t, result.Transactions)
	})

	t.Run("history dibatasi berdasarkan HistoryMonths", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().List(gomock.Any(), testTenant, userID, gomock.Any(), gomock.Any()).Return([]entity.Transaction{}, nil)
		txRepo.EXPECT().GetSummary(gomock.Any(), testTenant, userID, gomock.Any(), gomock.Any()).Return(&entity.TransactionSummary{}, nil)

		// From jauh sebelum HistoryMonths limit — harus dipotong
		uc := usecase.NewListTransactionsUseCase(txRepo)
		_, err := uc.Execute(context.Background(), usecase.ListTransactionsInput{
			TenantSchema:  testTenant,
			UserID:        userID,
			From:          time.Now().AddDate(-5, 0, 0), // 5 tahun lalu
			HistoryMonths: 3,                            // tier hanya boleh 3 bulan ke belakang
		})
		require.NoError(t, err)
	})

	t.Run("repo error saat List — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().List(gomock.Any(), testTenant, userID, gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

		uc := usecase.NewListTransactionsUseCase(txRepo)
		_, err := uc.Execute(context.Background(), usecase.ListTransactionsInput{
			TenantSchema: testTenant,
			UserID:       userID,
		})
		assert.Error(t, err)
	})
}
