package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListPaymentsUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — daftar payment dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		payRepo := mocks.NewMockPaymentReadRepository(ctrl)

		payments := []entity.PaymentSummary{
			{ID: uuid.New(), UserID: uuid.New(), PlanName: "PRO", AmountIDR: 150_000, Status: "PAID"},
		}
		payRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(payments, int64(1), nil)

		uc := usecase.NewListPaymentsUseCase(payRepo)
		out, err := uc.Execute(context.Background(), repository.PaymentListFilter{Limit: 10})
		require.NoError(t, err)
		assert.Len(t, out.Payments, 1)
		assert.Equal(t, int64(1), out.Total)
	})

	t.Run("daftar kosong — dikembalikan tanpa error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		payRepo := mocks.NewMockPaymentReadRepository(ctrl)

		payRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]entity.PaymentSummary{}, int64(0), nil)

		uc := usecase.NewListPaymentsUseCase(payRepo)
		out, err := uc.Execute(context.Background(), repository.PaymentListFilter{})
		require.NoError(t, err)
		assert.Empty(t, out.Payments)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		payRepo := mocks.NewMockPaymentReadRepository(ctrl)

		payRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("db error"))

		uc := usecase.NewListPaymentsUseCase(payRepo)
		_, err := uc.Execute(context.Background(), repository.PaymentListFilter{})
		assert.Error(t, err)
	})
}
