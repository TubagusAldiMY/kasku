package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/billing-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListPlansUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path returns plans ordered by price", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		expected := []entity.SubscriptionPlan{
			{ID: uuid.New(), Name: "FREE", PriceIDR: 0, IsActive: true},
			{ID: uuid.New(), Name: "BASIC", PriceIDR: 49000, IsActive: true},
			{ID: uuid.New(), Name: "PRO", PriceIDR: 99000, IsActive: true},
		}
		repo.EXPECT().ListAllPlans(gomock.Any()).Return(expected, nil)

		uc := usecase.NewListPlansUseCase(repo)
		got, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("repo error propagated wrapped", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		sentinel := errors.New("db down")
		repo.EXPECT().ListAllPlans(gomock.Any()).Return(nil, sentinel)

		uc := usecase.NewListPlansUseCase(repo)
		got, err := uc.Execute(context.Background())
		assert.Nil(t, got)
		assert.ErrorIs(t, err, sentinel)
	})
}
