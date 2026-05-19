package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDashboardStatsUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — semua stats dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		payRepo := mocks.NewMockPaymentReadRepository(ctrl)

		userRepo.EXPECT().CountTotal(gomock.Any()).Return(int64(1000), nil)
		userRepo.EXPECT().CountActive(gomock.Any()).Return(int64(800), nil)
		userRepo.EXPECT().CountCreatedSince(gomock.Any(), gomock.Any()).Return(int64(50), nil)
		payRepo.EXPECT().CountMRRActive(gomock.Any()).Return(int64(5_000_000), nil)
		payRepo.EXPECT().CountByTier(gomock.Any()).Return(map[string]int64{"PRO": 300, "BASIC": 500, "FREE": 200}, nil)
		payRepo.EXPECT().CountCancelledSince(gomock.Any(), gomock.Any()).Return(int64(20), nil)

		uc := usecase.NewDashboardStatsUseCase(userRepo, payRepo)
		stats, err := uc.Execute(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1000), stats.TotalUsers)
		assert.Equal(t, int64(5_000_000), stats.MRRIDR)
	})

	t.Run("CountTotal error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		payRepo := mocks.NewMockPaymentReadRepository(ctrl)

		userRepo.EXPECT().CountTotal(gomock.Any()).Return(int64(0), errors.New("db error"))

		uc := usecase.NewDashboardStatsUseCase(userRepo, payRepo)
		_, err := uc.Execute(context.Background())
		assert.Error(t, err)
	})

	t.Run("CountMRRActive error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRepo := mocks.NewMockUserReadRepository(ctrl)
		payRepo := mocks.NewMockPaymentReadRepository(ctrl)

		userRepo.EXPECT().CountTotal(gomock.Any()).Return(int64(100), nil)
		userRepo.EXPECT().CountActive(gomock.Any()).Return(int64(80), nil)
		userRepo.EXPECT().CountCreatedSince(gomock.Any(), gomock.Any()).Return(int64(5), nil)
		payRepo.EXPECT().CountMRRActive(gomock.Any()).Return(int64(0), errors.New("billing db error"))

		uc := usecase.NewDashboardStatsUseCase(userRepo, payRepo)
		_, err := uc.Execute(context.Background())
		assert.Error(t, err)
	})
}
