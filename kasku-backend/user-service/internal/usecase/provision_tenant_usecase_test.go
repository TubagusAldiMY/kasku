package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TubagusAldiMY/kasku/user-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/user-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestProvisionTenantUseCase_Execute(t *testing.T) {
	t.Parallel()

	log := zerolog.Nop()
	userID := uuid.New().String()
	email := "u***@example.com"
	username := "testuser"
	repoErr := errors.New("simulated db error")

	t.Run("happy path — semua repo berhasil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		financeRepo := mocks.NewMockFinanceRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		profileRepo := mocks.NewMockUserProfileRepository(ctrl)

		financeRepo.EXPECT().ProvisionTenant(gomock.Any(), userID).Return(nil)
		financeRepo.EXPECT().EnsureTenantRuntimeObjects(gomock.Any(), gomock.Any()).Return(nil)
		financeRepo.EXPECT().RemoveDefaultCategorySeeds(gomock.Any(), gomock.Any()).Return(nil)
		subRepo.EXPECT().CreateFreeSubscription(gomock.Any(), userID).Return(nil)
		profileRepo.EXPECT().EnsureUserProfile(gomock.Any(), userID, email, username).Return(nil)

		uc := usecase.NewProvisionTenantUseCase(financeRepo, subRepo, profileRepo, log)
		err := uc.Execute(context.Background(), userID, email, username)
		require.NoError(t, err)
	})

	t.Run("ProvisionTenant gagal — error propagate, repo lain tidak dipanggil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		financeRepo := mocks.NewMockFinanceRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		profileRepo := mocks.NewMockUserProfileRepository(ctrl)

		financeRepo.EXPECT().ProvisionTenant(gomock.Any(), userID).Return(repoErr)
		// Tidak ada EXPECT lain — gomock akan fail jika subRepo atau profileRepo dipanggil

		uc := usecase.NewProvisionTenantUseCase(financeRepo, subRepo, profileRepo, log)
		err := uc.Execute(context.Background(), userID, email, username)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provision tenant")
	})

	t.Run("EnsureTenantRuntimeObjects gagal — error propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		financeRepo := mocks.NewMockFinanceRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		profileRepo := mocks.NewMockUserProfileRepository(ctrl)

		financeRepo.EXPECT().ProvisionTenant(gomock.Any(), userID).Return(nil)
		financeRepo.EXPECT().EnsureTenantRuntimeObjects(gomock.Any(), gomock.Any()).Return(repoErr)

		uc := usecase.NewProvisionTenantUseCase(financeRepo, subRepo, profileRepo, log)
		err := uc.Execute(context.Background(), userID, email, username)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "runtime objects")
	})

	t.Run("CreateFreeSubscription gagal — error propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		financeRepo := mocks.NewMockFinanceRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		profileRepo := mocks.NewMockUserProfileRepository(ctrl)

		financeRepo.EXPECT().ProvisionTenant(gomock.Any(), userID).Return(nil)
		financeRepo.EXPECT().EnsureTenantRuntimeObjects(gomock.Any(), gomock.Any()).Return(nil)
		financeRepo.EXPECT().RemoveDefaultCategorySeeds(gomock.Any(), gomock.Any()).Return(nil)
		subRepo.EXPECT().CreateFreeSubscription(gomock.Any(), userID).Return(repoErr)

		uc := usecase.NewProvisionTenantUseCase(financeRepo, subRepo, profileRepo, log)
		err := uc.Execute(context.Background(), userID, email, username)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subscription FREE")
	})

	t.Run("EnsureUserProfile gagal — error propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)

		financeRepo := mocks.NewMockFinanceRepository(ctrl)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		profileRepo := mocks.NewMockUserProfileRepository(ctrl)

		financeRepo.EXPECT().ProvisionTenant(gomock.Any(), userID).Return(nil)
		financeRepo.EXPECT().EnsureTenantRuntimeObjects(gomock.Any(), gomock.Any()).Return(nil)
		financeRepo.EXPECT().RemoveDefaultCategorySeeds(gomock.Any(), gomock.Any()).Return(nil)
		subRepo.EXPECT().CreateFreeSubscription(gomock.Any(), userID).Return(nil)
		profileRepo.EXPECT().EnsureUserProfile(gomock.Any(), userID, email, username).Return(repoErr)

		uc := usecase.NewProvisionTenantUseCase(financeRepo, subRepo, profileRepo, log)
		err := uc.Execute(context.Background(), userID, email, username)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user profile")
	})
}
