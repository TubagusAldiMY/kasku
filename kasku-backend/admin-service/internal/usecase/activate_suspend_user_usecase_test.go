package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// ─── ActivateUser ──────────────────────────────────────────────────────────────

func TestActivateUserUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — user diaktifkan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRead := mocks.NewMockUserReadRepository(ctrl)
		userWrite := mocks.NewMockUserWriteRepository(ctrl)
		auditLogger, mockAudit := testAuditLogger(ctrl)
		userID := uuid.New()

		user := &entity.UserSummary{ID: userID, Email: "user@example.com", IsActive: false, CreatedAt: time.Now()}
		userRead.EXPECT().GetByID(gomock.Any(), userID).Return(user, nil)
		userWrite.EXPECT().SetIsActive(gomock.Any(), userID, true).Return(nil)
		mockAudit.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		uc := usecase.NewActivateUserUseCase(userRead, userWrite, auditLogger)
		err := uc.Execute(context.Background(), usecase.ActivateUserInput{
			AdminID:      uuid.New(),
			TargetUserID: userID,
			Reason:       "request dari user",
		})
		require.NoError(t, err)
	})

	t.Run("user tidak ditemukan — ErrUserNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRead := mocks.NewMockUserReadRepository(ctrl)
		userWrite := mocks.NewMockUserWriteRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)
		userID := uuid.New()

		userRead.EXPECT().GetByID(gomock.Any(), userID).Return(nil, nil)

		uc := usecase.NewActivateUserUseCase(userRead, userWrite, auditLogger)
		err := uc.Execute(context.Background(), usecase.ActivateUserInput{
			AdminID:      uuid.New(),
			TargetUserID: userID,
		})
		assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	})

	t.Run("repo error saat GetByID — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRead := mocks.NewMockUserReadRepository(ctrl)
		userWrite := mocks.NewMockUserWriteRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)
		userID := uuid.New()

		userRead.EXPECT().GetByID(gomock.Any(), userID).Return(nil, errors.New("db error"))

		uc := usecase.NewActivateUserUseCase(userRead, userWrite, auditLogger)
		err := uc.Execute(context.Background(), usecase.ActivateUserInput{AdminID: uuid.New(), TargetUserID: userID})
		assert.Error(t, err)
	})
}

// ─── SuspendUser ──────────────────────────────────────────────────────────────

func TestSuspendUserUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — user disuspend", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRead := mocks.NewMockUserReadRepository(ctrl)
		userWrite := mocks.NewMockUserWriteRepository(ctrl)
		auditLogger, mockAudit := testAuditLogger(ctrl)
		userID := uuid.New()

		user := &entity.UserSummary{ID: userID, Email: "user@example.com", IsActive: true, CreatedAt: time.Now()}
		userRead.EXPECT().GetByID(gomock.Any(), userID).Return(user, nil)
		userWrite.EXPECT().SetIsActive(gomock.Any(), userID, false).Return(nil)
		mockAudit.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		uc := usecase.NewSuspendUserUseCase(userRead, userWrite, auditLogger)
		err := uc.Execute(context.Background(), usecase.SuspendUserInput{
			AdminID:      uuid.New(),
			TargetUserID: userID,
			Reason:       "pelanggaran TOS",
		})
		require.NoError(t, err)
	})

	t.Run("user tidak ditemukan — ErrUserNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		userRead := mocks.NewMockUserReadRepository(ctrl)
		userWrite := mocks.NewMockUserWriteRepository(ctrl)
		auditLogger, _ := testAuditLogger(ctrl)
		userID := uuid.New()

		userRead.EXPECT().GetByID(gomock.Any(), userID).Return(nil, nil)

		uc := usecase.NewSuspendUserUseCase(userRead, userWrite, auditLogger)
		err := uc.Execute(context.Background(), usecase.SuspendUserInput{AdminID: uuid.New(), TargetUserID: userID})
		assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	})
}
