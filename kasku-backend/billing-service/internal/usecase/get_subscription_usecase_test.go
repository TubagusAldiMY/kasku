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

func TestGetSubscriptionUseCase_Execute(t *testing.T) {
	t.Parallel()
	userID := uuid.NewString()

	t.Run("found returns subscription", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		want := &entity.Subscription{ID: uuid.New(), UserID: uuid.MustParse(userID), Status: entity.StatusActive}
		repo.EXPECT().GetByUserID(gomock.Any(), userID).Return(want, nil)

		uc := usecase.NewGetSubscriptionUseCase(repo)
		got, err := uc.Execute(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("not found propagated", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		repo.EXPECT().GetByUserID(gomock.Any(), userID).Return(nil, domainerrors.ErrSubscriptionNotFound)

		uc := usecase.NewGetSubscriptionUseCase(repo)
		got, err := uc.Execute(context.Background(), userID)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, domainerrors.ErrSubscriptionNotFound)
	})

	t.Run("other repo error propagated wrapped", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockSubscriptionRepository(ctrl)
		sentinel := errors.New("boom")
		repo.EXPECT().GetByUserID(gomock.Any(), userID).Return(nil, sentinel)

		uc := usecase.NewGetSubscriptionUseCase(repo)
		_, err := uc.Execute(context.Background(), userID)
		assert.ErrorIs(t, err, sentinel)
	})
}
