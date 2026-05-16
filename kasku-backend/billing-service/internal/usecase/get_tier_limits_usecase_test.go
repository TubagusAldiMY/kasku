package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/billing-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetTierLimitsUseCase_Execute(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	planID := uuid.New()

	t.Run("active subscription with active plan returns plan limits", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().GetByUserID(gomock.Any(), userID.String()).Return(&entity.Subscription{
			ID: uuid.New(), UserID: userID, PlanID: planID, Status: entity.StatusActive,
		}, nil)
		plan := &entity.SubscriptionPlan{
			ID: planID, Name: "BASIC", PriceIDR: 49000, IsActive: true,
			Limits: entity.PlanLimits{
				MaxTransactionsPerMonth:   500,
				MaxFinancialAccounts:      10,
				MaxInvestmentInstruments:  5,
				HistoryRetentionMonths:    12,
				EmailNotificationsEnabled: true,
				ExportCsvEnabled:          true,
			},
		}
		repo.EXPECT().GetPlanWithLimits(gomock.Any(), planID.String()).Return(plan, nil)

		uc := usecase.NewGetTierLimitsUseCase(repo)
		got, err := uc.Execute(context.Background(), userID.String())
		require.NoError(t, err)
		assert.Equal(t, &plan.Limits, got)
	})

	t.Run("no subscription returns FREE tier fallback", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().GetByUserID(gomock.Any(), userID.String()).Return(nil, domainerrors.ErrSubscriptionNotFound)

		uc := usecase.NewGetTierLimitsUseCase(repo)
		got, err := uc.Execute(context.Background(), userID.String())
		require.NoError(t, err)
		require.NotNil(t, got)
		// Hard guard: harus sinkron dengan seed FREE di migration 000002.
		assert.Equal(t, int32(50), got.MaxTransactionsPerMonth)
		assert.Equal(t, int32(3), got.MaxFinancialAccounts)
		assert.False(t, got.ExportCsvEnabled)
	})

	t.Run("plan inactive returns FREE tier fallback", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().GetByUserID(gomock.Any(), userID.String()).Return(&entity.Subscription{
			ID: uuid.New(), UserID: userID, PlanID: planID, Status: entity.StatusActive,
		}, nil)
		repo.EXPECT().GetPlanWithLimits(gomock.Any(), planID.String()).Return(nil, domainerrors.ErrPlanNotFound)

		uc := usecase.NewGetTierLimitsUseCase(repo)
		got, err := uc.Execute(context.Background(), userID.String())
		require.NoError(t, err)
		assert.Equal(t, int32(50), got.MaxTransactionsPerMonth)
	})

	t.Run("other repo error propagated wrapped", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		sentinel := errors.New("boom")
		repo.EXPECT().GetByUserID(gomock.Any(), userID.String()).Return(nil, sentinel)

		uc := usecase.NewGetTierLimitsUseCase(repo)
		_, err := uc.Execute(context.Background(), userID.String())
		assert.ErrorIs(t, err, sentinel)
	})
}
