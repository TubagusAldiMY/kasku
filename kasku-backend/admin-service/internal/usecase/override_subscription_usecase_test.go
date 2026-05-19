package usecase_test

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestOverrideSubscriptionUseCase_Execute(t *testing.T) {
	t.Parallel()

	makeInput := func(reason string) usecase.OverrideSubscriptionInput {
		return usecase.OverrideSubscriptionInput{
			AdminID:      uuid.New(),
			TargetUserID: uuid.New(),
			NewPlanName:  "PRO",
			Reason:       reason,
		}
	}

	t.Run("happy path — plan diubah", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		auditLogger, mockAudit := testAuditLogger(ctrl)
		in := makeInput("admin request")

		currentSub := &repository.SubscriptionView{ID: uuid.New(), UserID: in.TargetUserID, PlanName: "FREE"}
		newPlanID := uuid.New()

		subRepo.EXPECT().GetByUserID(gomock.Any(), in.TargetUserID).Return(currentSub, nil)
		subRepo.EXPECT().FindPlanByName(gomock.Any(), "PRO").Return(newPlanID, int64(99_000), nil)
		subRepo.EXPECT().UpdatePlan(gomock.Any(), currentSub.ID, newPlanID, gomock.Any()).Return(nil)
		mockAudit.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		uc := usecase.NewOverrideSubscriptionUseCase(subRepo, auditLogger)
		err := uc.Execute(context.Background(), in)
		require.NoError(t, err)
	})

	t.Run("reason kosong — ErrValidation", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)

		uc := usecase.NewOverrideSubscriptionUseCase(subRepo, auditLogger)
		err := uc.Execute(context.Background(), makeInput("  "))
		assert.ErrorIs(t, err, domainerrors.ErrValidation)
	})

	t.Run("subscription tidak ditemukan — ErrSubscriptionNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)
		in := makeInput("valid reason")

		subRepo.EXPECT().GetByUserID(gomock.Any(), in.TargetUserID).Return(nil, nil)

		uc := usecase.NewOverrideSubscriptionUseCase(subRepo, auditLogger)
		err := uc.Execute(context.Background(), in)
		assert.ErrorIs(t, err, domainerrors.ErrSubscriptionNotFound)
	})

	t.Run("plan tidak ditemukan — ErrPlanNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)
		in := makeInput("valid reason")

		currentSub := &repository.SubscriptionView{ID: uuid.New(), UserID: in.TargetUserID, PlanName: "FREE"}

		subRepo.EXPECT().GetByUserID(gomock.Any(), in.TargetUserID).Return(currentSub, nil)
		subRepo.EXPECT().FindPlanByName(gomock.Any(), "PRO").Return(uuid.Nil, int64(0), nil)

		uc := usecase.NewOverrideSubscriptionUseCase(subRepo, auditLogger)
		err := uc.Execute(context.Background(), in)
		assert.ErrorIs(t, err, domainerrors.ErrPlanNotFound)
	})

	t.Run("repo error saat GetByUserID — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		subRepo := mocks.NewMockSubscriptionRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)
		in := makeInput("valid reason")

		subRepo.EXPECT().GetByUserID(gomock.Any(), in.TargetUserID).Return(nil, errors.New("db error"))

		uc := usecase.NewOverrideSubscriptionUseCase(subRepo, auditLogger)
		err := uc.Execute(context.Background(), in)
		assert.Error(t, err)
	})
}
