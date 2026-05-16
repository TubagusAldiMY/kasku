package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/messaging"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/billing-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestExpireSubscriptionsUseCase_Execute(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()

	t.Run("empty list returns 0", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().ListExpiredSubscriptions(gomock.Any()).Return(nil, nil)

		uc := usecase.NewExpireSubscriptionsUseCase(repo, logger)
		n, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("multiple subscriptions success with cached plan name", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		planID := uuid.New()
		sub1 := entity.Subscription{ID: uuid.New(), UserID: uuid.New(), PlanID: planID, Status: entity.StatusActive}
		sub2 := entity.Subscription{ID: uuid.New(), UserID: uuid.New(), PlanID: planID, Status: entity.StatusActive}

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().ListExpiredSubscriptions(gomock.Any()).Return([]entity.Subscription{sub1, sub2}, nil)
		// Plan name lookup hanya 1x karena di-cache.
		repo.EXPECT().GetPlanWithLimits(gomock.Any(), planID.String()).
			Return(&entity.SubscriptionPlan{ID: planID, Name: "BASIC", IsActive: true}, nil).
			Times(1)
		// Dua kali ExpireSubscriptionAtomic dengan payload event yang benar.
		repo.EXPECT().ExpireSubscriptionAtomic(
			gomock.Any(),
			gomock.Any(),
			gomock.Eq("subscription.expired"),
			gomock.Eq(messaging.RoutingKeySubscriptionExpired),
			gomock.AssignableToTypeOf([]byte{}),
		).DoAndReturn(func(_ context.Context, subID, _, _ string, payload []byte) (bool, error) {
			var got messaging.SubscriptionExpiredEvent
			if err := json.Unmarshal(payload, &got); err != nil {
				return false, err
			}
			assert.Equal(t, subID, got.SubscriptionID)
			assert.Equal(t, "BASIC", got.PlanName)
			assert.Equal(t, "ACTIVE", got.PreviousStatus)
			assert.NotEmpty(t, got.ExpiredAt)
			return true, nil
		}).Times(2)

		uc := usecase.NewExpireSubscriptionsUseCase(repo, logger)
		n, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 2, n)
	})

	t.Run("flipped=false counts as skipped (idempotent re-run)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		planID := uuid.New()
		sub := entity.Subscription{ID: uuid.New(), UserID: uuid.New(), PlanID: planID, Status: entity.StatusActive}

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().ListExpiredSubscriptions(gomock.Any()).Return([]entity.Subscription{sub}, nil)
		repo.EXPECT().GetPlanWithLimits(gomock.Any(), planID.String()).
			Return(&entity.SubscriptionPlan{ID: planID, Name: "BASIC", IsActive: true}, nil)
		repo.EXPECT().ExpireSubscriptionAtomic(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(false, nil)

		uc := usecase.NewExpireSubscriptionsUseCase(repo, logger)
		n, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("partial failure does not stop loop", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		planID := uuid.New()
		subBad := entity.Subscription{ID: uuid.New(), UserID: uuid.New(), PlanID: planID, Status: entity.StatusActive}
		subGood := entity.Subscription{ID: uuid.New(), UserID: uuid.New(), PlanID: planID, Status: entity.StatusActive}

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().ListExpiredSubscriptions(gomock.Any()).Return([]entity.Subscription{subBad, subGood}, nil)
		repo.EXPECT().GetPlanWithLimits(gomock.Any(), planID.String()).
			Return(&entity.SubscriptionPlan{ID: planID, Name: "BASIC", IsActive: true}, nil).Times(1)

		gomock.InOrder(
			repo.EXPECT().ExpireSubscriptionAtomic(
				gomock.Any(), gomock.Eq(subBad.ID.String()), gomock.Any(), gomock.Any(), gomock.Any(),
			).Return(false, errors.New("tx error")),
			repo.EXPECT().ExpireSubscriptionAtomic(
				gomock.Any(), gomock.Eq(subGood.ID.String()), gomock.Any(), gomock.Any(), gomock.Any(),
			).Return(true, nil),
		)

		uc := usecase.NewExpireSubscriptionsUseCase(repo, logger)
		n, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, n)
	})

	t.Run("plan not found skips that subscription", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		planID := uuid.New()
		sub := entity.Subscription{ID: uuid.New(), UserID: uuid.New(), PlanID: planID, Status: entity.StatusActive}

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().ListExpiredSubscriptions(gomock.Any()).Return([]entity.Subscription{sub}, nil)
		repo.EXPECT().GetPlanWithLimits(gomock.Any(), planID.String()).Return(nil, domainerrors.ErrPlanNotFound)
		// ExpireSubscriptionAtomic tidak dipanggil karena plan tidak ditemukan.

		uc := usecase.NewExpireSubscriptionsUseCase(repo, logger)
		n, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("ListExpiredSubscriptions error propagated wrapped", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		sentinel := errors.New("boom")
		repo.EXPECT().ListExpiredSubscriptions(gomock.Any()).Return(nil, sentinel)

		uc := usecase.NewExpireSubscriptionsUseCase(repo, logger)
		n, err := uc.Execute(context.Background())
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, sentinel)
	})
}
