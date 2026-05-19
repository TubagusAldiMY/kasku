package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/finance-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDeleteAccountUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — SoftDelete berhasil", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()

		repo.EXPECT().SoftDelete(gomock.Any(), testTenant, accountID, userID).Return(nil)

		uc := usecase.NewDeleteAccountUseCase(repo)
		err := uc.Execute(context.Background(), testTenant, accountID, userID)
		require.NoError(t, err)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()

		repo.EXPECT().SoftDelete(gomock.Any(), testTenant, accountID, userID).Return(errors.New("db error"))

		uc := usecase.NewDeleteAccountUseCase(repo)
		err := uc.Execute(context.Background(), testTenant, accountID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hapus akun")
	})
}
