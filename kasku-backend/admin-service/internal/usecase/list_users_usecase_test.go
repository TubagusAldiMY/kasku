package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListUsersUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — daftar user dan tier dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)

		userID := uuid.New()
		users := []entity.UserSummary{
			{ID: userID, Email: "user@example.com", CreatedAt: time.Now()},
		}
		subs := map[uuid.UUID]repository.SubscriptionView{
			userID: {PlanName: "PRO", Status: "ACTIVE"},
		}

		userRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(users, int64(1), nil)
		subRepo.EXPECT().GetByUserIDs(gomock.Any(), gomock.Any()).Return(subs, nil)

		uc := usecase.NewListUsersUseCase(userRepo, subRepo)
		out, err := uc.Execute(context.Background(), usecase.ListUsersInput{Filter: repository.UserListFilter{Limit: 10}})
		require.NoError(t, err)
		assert.Len(t, out.Users, 1)
		assert.Equal(t, "PRO", out.Users[0].SubscriptionTier)
	})

	t.Run("daftar kosong — GetByUserIDs tidak dipanggil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)

		userRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]entity.UserSummary{}, int64(0), nil)

		uc := usecase.NewListUsersUseCase(userRepo, subRepo)
		out, err := uc.Execute(context.Background(), usecase.ListUsersInput{})
		require.NoError(t, err)
		assert.Empty(t, out.Users)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)

		userRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("db error"))

		uc := usecase.NewListUsersUseCase(userRepo, subRepo)
		_, err := uc.Execute(context.Background(), usecase.ListUsersInput{})
		assert.Error(t, err)
	})
}
