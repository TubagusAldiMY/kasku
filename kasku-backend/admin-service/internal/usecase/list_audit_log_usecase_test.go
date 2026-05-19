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

func TestListAuditLogUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — entries dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		auditRepo := mocks.NewMockAuditLogRepository(ctrl)

		entries := []entity.AuditLogEntry{
			{ID: uuid.New(), AdminID: uuid.New(), Action: entity.AuditActionLogin, Success: true, CreatedAt: time.Now()},
		}
		auditRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(entries, int64(1), nil)

		uc := usecase.NewListAuditLogUseCase(auditRepo)
		out, err := uc.Execute(context.Background(), repository.AuditLogFilter{Limit: 10})
		require.NoError(t, err)
		assert.Len(t, out.Entries, 1)
		assert.Equal(t, int64(1), out.Total)
	})

	t.Run("daftar kosong — dikembalikan tanpa error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		auditRepo := mocks.NewMockAuditLogRepository(ctrl)

		auditRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]entity.AuditLogEntry{}, int64(0), nil)

		uc := usecase.NewListAuditLogUseCase(auditRepo)
		out, err := uc.Execute(context.Background(), repository.AuditLogFilter{})
		require.NoError(t, err)
		assert.Empty(t, out.Entries)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		auditRepo := mocks.NewMockAuditLogRepository(ctrl)

		auditRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("db error"))

		uc := usecase.NewListAuditLogUseCase(auditRepo)
		_, err := uc.Execute(context.Background(), repository.AuditLogFilter{})
		assert.Error(t, err)
	})
}
