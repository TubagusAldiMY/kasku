package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/finance-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUpdateAccountUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — akun terupdate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New()

		existing := &entity.FinancialAccount{
			ID: accountID, UserID: uuid.MustParse(userID), Name: "Old Name", CreatedAt: time.Now(),
		}
		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID.String(), userID).Return(existing, nil)
		repo.EXPECT().Update(gomock.Any(), testTenant, gomock.Any()).Return(nil)

		uc := usecase.NewUpdateAccountUseCase(repo)
		input := usecase.UpdateAccountInput{
			TenantSchema: testTenant,
			ID:           accountID.String(),
			UserID:       userID,
			Name:         "New Name",
		}
		acc, err := uc.Execute(context.Background(), input)
		require.NoError(t, err)
		assert.Equal(t, "New Name", acc.Name)
	})

	t.Run("nama kosong — ErrInvalidInput", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)

		uc := usecase.NewUpdateAccountUseCase(repo)
		_, err := uc.Execute(context.Background(), usecase.UpdateAccountInput{
			TenantSchema: testTenant, ID: uuid.New().String(), UserID: testUserID(), Name: "",
		})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})

	t.Run("akun tidak ditemukan — GetByID nil mengembalikan ErrAccountNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()

		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID, userID).Return(nil, nil)

		uc := usecase.NewUpdateAccountUseCase(repo)
		_, err := uc.Execute(context.Background(), usecase.UpdateAccountInput{
			TenantSchema: testTenant, ID: accountID, UserID: userID, Name: "X",
		})
		assert.ErrorIs(t, err, domainerrors.ErrAccountNotFound)
	})

	t.Run("repo error saat GetByID — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFinancialAccountRepository(ctrl)
		userID := testUserID()
		accountID := uuid.New().String()

		repo.EXPECT().GetByID(gomock.Any(), testTenant, accountID, userID).Return(nil, errors.New("db error"))

		uc := usecase.NewUpdateAccountUseCase(repo)
		_, err := uc.Execute(context.Background(), usecase.UpdateAccountInput{
			TenantSchema: testTenant, ID: accountID, UserID: userID, Name: "Test",
		})
		assert.Error(t, err)
	})
}
