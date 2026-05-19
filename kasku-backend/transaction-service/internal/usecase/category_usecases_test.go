package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/transaction-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// ─── ListCategories ───────────────────────────────────────────────────────────

func TestListCategoriesUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — daftar kategori dikembalikan", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)

		cats := []entity.Category{
			{ID: uuid.New(), Name: "Makan", CategoryType: entity.CategoryExpense},
			{ID: uuid.New(), Name: "Gaji", CategoryType: entity.CategoryIncome},
		}
		catRepo.EXPECT().List(gomock.Any(), testTenant).Return(cats, nil)

		uc := usecase.NewListCategoriesUseCase(catRepo)
		result, err := uc.Execute(context.Background(), testTenant)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("repo error — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)

		catRepo.EXPECT().List(gomock.Any(), testTenant).Return(nil, errors.New("db error"))

		uc := usecase.NewListCategoriesUseCase(catRepo)
		_, err := uc.Execute(context.Background(), testTenant)
		assert.Error(t, err)
	})
}

// ─── CreateCategory ───────────────────────────────────────────────────────────

func TestCreateCategoryUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — kategori dibuat dengan default icon dan color", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)

		catRepo.EXPECT().Create(gomock.Any(), testTenant, gomock.Any()).Return(nil)

		uc := usecase.NewCreateCategoryUseCase(catRepo)
		cat, err := uc.Execute(context.Background(), usecase.CreateCategoryInput{
			TenantSchema: testTenant,
			Name:         "Transport",
			CategoryType: entity.CategoryExpense,
			// Icon dan Color kosong — harus diisi default
		})
		require.NoError(t, err)
		assert.Equal(t, "tag", cat.Icon)
		assert.Equal(t, "#6366f1", cat.Color)
	})

	t.Run("nama kosong — ErrInvalidInput", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)

		uc := usecase.NewCreateCategoryUseCase(catRepo)
		_, err := uc.Execute(context.Background(), usecase.CreateCategoryInput{
			TenantSchema: testTenant, Name: "", CategoryType: entity.CategoryExpense,
		})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})

	t.Run("category_type tidak valid — ErrInvalidInput", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)

		uc := usecase.NewCreateCategoryUseCase(catRepo)
		_, err := uc.Execute(context.Background(), usecase.CreateCategoryInput{
			TenantSchema: testTenant, Name: "Test", CategoryType: "INVALID",
		})
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})
}

// ─── UpdateCategory ───────────────────────────────────────────────────────────

func TestUpdateCategoryUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — kategori diupdate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		catID := uuid.New().String()

		existing := &entity.Category{
			ID: uuid.MustParse(catID), Name: "Lama",
			CategoryType: entity.CategoryExpense, CreatedAt: time.Now(),
		}
		catRepo.EXPECT().GetByID(gomock.Any(), testTenant, catID).Return(existing, nil)
		catRepo.EXPECT().Update(gomock.Any(), testTenant, gomock.Any()).Return(nil)

		uc := usecase.NewUpdateCategoryUseCase(catRepo)
		err := uc.Execute(context.Background(), testTenant, catID, "Baru", "tag", "#fff", entity.CategoryBoth)
		require.NoError(t, err)
	})

	t.Run("nama kosong — ErrInvalidInput", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)

		uc := usecase.NewUpdateCategoryUseCase(catRepo)
		err := uc.Execute(context.Background(), testTenant, uuid.New().String(), "", "tag", "#fff", entity.CategoryExpense)
		assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	})
}

// ─── DeleteCategory ───────────────────────────────────────────────────────────

func TestDeleteCategoryUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy path — kategori tanpa transaksi aktif dihapus", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		catID := uuid.New()

		existing := &entity.Category{ID: catID, Name: "Transport", CategoryType: entity.CategoryExpense, IsDefault: false}
		catRepo.EXPECT().GetByID(gomock.Any(), testTenant, catID.String()).Return(existing, nil)
		catRepo.EXPECT().HasActiveTransactions(gomock.Any(), testTenant, catID.String()).Return(false, nil)
		catRepo.EXPECT().SoftDelete(gomock.Any(), testTenant, catID.String()).Return(nil)

		uc := usecase.NewDeleteCategoryUseCase(catRepo)
		err := uc.Execute(context.Background(), testTenant, catID.String())
		require.NoError(t, err)
	})

	t.Run("kategori tidak ditemukan — ErrCategoryNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		catID := uuid.New().String()

		catRepo.EXPECT().GetByID(gomock.Any(), testTenant, catID).Return(nil, domainerrors.ErrCategoryNotFound)

		uc := usecase.NewDeleteCategoryUseCase(catRepo)
		err := uc.Execute(context.Background(), testTenant, catID)
		assert.ErrorIs(t, err, domainerrors.ErrCategoryNotFound)
	})

	t.Run("kategori default — ErrDefaultCategoryCannotBeDeleted", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		catID := uuid.New()

		existing := &entity.Category{ID: catID, Name: "Gaji", CategoryType: entity.CategoryIncome, IsDefault: true}
		catRepo.EXPECT().GetByID(gomock.Any(), testTenant, catID.String()).Return(existing, nil)

		uc := usecase.NewDeleteCategoryUseCase(catRepo)
		err := uc.Execute(context.Background(), testTenant, catID.String())
		assert.ErrorIs(t, err, domainerrors.ErrDefaultCategoryCannotBeDeleted)
	})

	t.Run("kategori punya transaksi aktif — ErrCategoryHasTransactions", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		catID := uuid.New()

		existing := &entity.Category{ID: catID, Name: "Makan", CategoryType: entity.CategoryExpense, IsDefault: false}
		catRepo.EXPECT().GetByID(gomock.Any(), testTenant, catID.String()).Return(existing, nil)
		catRepo.EXPECT().HasActiveTransactions(gomock.Any(), testTenant, catID.String()).Return(true, nil)

		uc := usecase.NewDeleteCategoryUseCase(catRepo)
		err := uc.Execute(context.Background(), testTenant, catID.String())
		assert.ErrorIs(t, err, domainerrors.ErrCategoryHasTransactions)
	})

	t.Run("repo error saat HasActiveTransactions — propagate", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		catRepo := mocks.NewMockCategoryRepository(ctrl)
		catID := uuid.New()

		existing := &entity.Category{ID: catID, Name: "Tagihan", CategoryType: entity.CategoryExpense, IsDefault: false}
		catRepo.EXPECT().GetByID(gomock.Any(), testTenant, catID.String()).Return(existing, nil)
		catRepo.EXPECT().HasActiveTransactions(gomock.Any(), testTenant, catID.String()).Return(false, errors.New("db error"))

		uc := usecase.NewDeleteCategoryUseCase(catRepo)
		err := uc.Execute(context.Background(), testTenant, catID.String())
		assert.Error(t, err)
	})
}
