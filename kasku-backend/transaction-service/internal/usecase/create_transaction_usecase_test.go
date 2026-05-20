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

func TestCreateTransactionUseCase_Execute(t *testing.T) {
	t.Parallel()

	baseInput := func(userID, accountID string) usecase.CreateTransactionInput {
		return usecase.CreateTransactionInput{
			TenantSchema:    testTenant,
			UserID:          userID,
			AccountID:       accountID,
			TransactionType: entity.TransactionExpense,
			AmountIDR:       50_000,
			TransactionDate: time.Now().UTC(),
			MaxTransactions: -1, // unlimited
		}
	}

	t.Run("happy path — transaksi berhasil dibuat", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		userID := testUserID()
		accountID := testAccountID()

		txRepo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(nil)

		uc := usecase.NewCreateTransactionUseCase(txRepo, catRepo)
		tx, err := uc.Execute(context.Background(), baseInput(userID, accountID))
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, tx.ID)
		assert.Equal(t, int64(50_000), tx.AmountIDR)
		assert.NotEmpty(t, tx.SyncID) // auto-generated
	})

	t.Run("amount nol atau negatif — ErrInvalidInput", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		userID := testUserID()

		uc := usecase.NewCreateTransactionUseCase(txRepo, catRepo)
		input := baseInput(userID, testAccountID())
		input.AmountIDR = 0
		_, err := uc.Execute(context.Background(), input)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})

	t.Run("limit bulanan tercapai — ErrTransactionLimitReached", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().CountMonthly(gomock.Any(), testTenant, userID, gomock.Any()).Return(50, nil)

		uc := usecase.NewCreateTransactionUseCase(txRepo, catRepo)
		input := baseInput(userID, testAccountID())
		input.MaxTransactions = 50 // sudah di batas
		_, err := uc.Execute(context.Background(), input)
		assert.ErrorIs(t, err, domainerrors.ErrTransactionLimitReached)
	})

	t.Run("limit -1 (unlimited) — CountMonthly tidak dipanggil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(nil)
		// CountMonthly tidak boleh dipanggil

		uc := usecase.NewCreateTransactionUseCase(txRepo, catRepo)
		input := baseInput(userID, testAccountID())
		input.MaxTransactions = -1
		_, err := uc.Execute(context.Background(), input)
		require.NoError(t, err)
	})

	t.Run("sync_id kosong — di-generate otomatis", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(nil)

		uc := usecase.NewCreateTransactionUseCase(txRepo, catRepo)
		input := baseInput(userID, testAccountID())
		input.SyncID = "" // kosong — harus di-generate
		tx, err := uc.Execute(context.Background(), input)
		require.NoError(t, err)
		_, parseErr := uuid.Parse(tx.SyncID)
		assert.NoError(t, parseErr, "SyncID harus berupa UUID yang valid")
	})

	t.Run("AccountID tidak valid UUID — tidak panic, disimpan sebagai zero UUID", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(nil)

		uc := usecase.NewCreateTransactionUseCase(txRepo, catRepo)
		input := baseInput(userID, "bukan-uuid")
		tx, err := uc.Execute(context.Background(), input)
		require.NoError(t, err)
		assert.Equal(t, uuid.Nil, tx.AccountID) // invalid UUID → zero value
	})

	t.Run("repo error saat Create — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		userID := testUserID()

		txRepo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(errors.New("db error"))

		uc := usecase.NewCreateTransactionUseCase(txRepo, catRepo)
		_, err := uc.Execute(context.Background(), baseInput(userID, testAccountID()))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gagal buat transaksi")
	})

	t.Run("TRANSFER saldo tidak cukup — ErrInsufficientBalance dipropagasi", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		txRepo := mocks.NewMockTransactionRepository(ctrl)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		userID := testUserID()

		// Repository mengembalikan ErrInsufficientBalance (dicek di dalam DB transaction).
		txRepo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(domainerrors.ErrInsufficientBalance)

		uc := usecase.NewCreateTransactionUseCase(txRepo, catRepo)
		input := baseInput(userID, testAccountID())
		input.TransactionType = entity.TransactionTransfer
		input.ToAccountID = testAccountID()
		input.AmountIDR = 999_999_999

		_, err := uc.Execute(context.Background(), input)
		assert.ErrorIs(t, err, domainerrors.ErrInsufficientBalance)
	})
}
