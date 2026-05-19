package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/transaction-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestExportCSVUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("export tidak diizinkan — ErrExportNotAllowed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)

		uc := usecase.NewExportCSVUseCase(txRepo)
		_, err := uc.Execute(context.Background(), testTenant, testUserID(), false)
		assert.ErrorIs(t, err, domainerrors.ErrExportNotAllowed)
	})

	t.Run("happy path — CSV berisi header dan data", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()

		txs := []entity.Transaction{
			{
				ID: uuid.New(), SyncID: "sync-1",
				AccountID:       uuid.New(),
				TransactionType: entity.TransactionIncome,
				AmountIDR:       200_000,
				TransactionDate: time.Now().UTC(),
				Notes:           "gaji",
				CreatedAt:       time.Now().UTC(),
			},
		}
		txRepo.EXPECT().ListForExport(gomock.Any(), testTenant, userID, nil, nil).Return(txs, nil)

		uc := usecase.NewExportCSVUseCase(txRepo)
		data, err := uc.Execute(context.Background(), testTenant, userID, true)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Contains(t, string(data), "INCOME")
		assert.Contains(t, string(data), "200000")
	})

	t.Run("data kosong — CSV hanya berisi header", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().ListForExport(gomock.Any(), testTenant, userID, nil, nil).Return([]entity.Transaction{}, nil)

		uc := usecase.NewExportCSVUseCase(txRepo)
		data, err := uc.Execute(context.Background(), testTenant, userID, true)
		require.NoError(t, err)
		assert.Contains(t, string(data), "ID") // header harus ada
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().ListForExport(gomock.Any(), testTenant, userID, nil, nil).Return(nil, errors.New("db error"))

		uc := usecase.NewExportCSVUseCase(txRepo)
		_, err := uc.Execute(context.Background(), testTenant, userID, true)
		assert.Error(t, err)
	})
}
