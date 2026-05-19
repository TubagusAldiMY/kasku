package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetUserDetailUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — user dan subscription dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		userID := uuid.New()

		summary := &entity.UserSummary{ID: userID, Email: "user@example.com", CreatedAt: time.Now()}
		subView := &repository.SubscriptionView{PlanName: "BASIC", Status: "ACTIVE", CurrentPeriodStart: time.Now()}

		userRepo.EXPECT().GetByID(gomock.Any(), userID).Return(summary, nil)
		subRepo.EXPECT().GetByUserID(gomock.Any(), userID).Return(subView, nil)

		uc := usecase.NewGetUserDetailUseCase(userRepo, subRepo)
		detail, err := uc.Execute(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, "BASIC", detail.SubscriptionTier)
	})

	t.Run("user tidak ditemukan — ErrUserNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		userID := uuid.New()

		userRepo.EXPECT().GetByID(gomock.Any(), userID).Return(nil, nil)

		uc := usecase.NewGetUserDetailUseCase(userRepo, subRepo)
		_, err := uc.Execute(context.Background(), userID)
		assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		userID := uuid.New()

		userRepo.EXPECT().GetByID(gomock.Any(), userID).Return(nil, errors.New("db error"))

		uc := usecase.NewGetUserDetailUseCase(userRepo, subRepo)
		_, err := uc.Execute(context.Background(), userID)
		assert.Error(t, err)
	})
}
