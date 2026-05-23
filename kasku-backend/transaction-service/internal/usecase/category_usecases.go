package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
	"github.com/google/uuid"
)

type ListCategoriesUseCase struct {
	catRepo repository.CategoryRepository
}

func NewListCategoriesUseCase(catRepo repository.CategoryRepository) *ListCategoriesUseCase {
	return &ListCategoriesUseCase{catRepo: catRepo}
}

func (uc *ListCategoriesUseCase) Execute(ctx context.Context, tenantSchema string) ([]entity.Category, error) {
	cats, err := uc.catRepo.List(ctx, tenantSchema)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil daftar kategori: %w", err)
	}
	return cats, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type CreateCategoryInput struct {
	TenantSchema string
	Name         string
	Icon         string
	Color        string
	CategoryType entity.CategoryType
}

type CreateCategoryUseCase struct {
	catRepo repository.CategoryRepository
}

func NewCreateCategoryUseCase(catRepo repository.CategoryRepository) *CreateCategoryUseCase {
	return &CreateCategoryUseCase{catRepo: catRepo}
}

func (uc *CreateCategoryUseCase) Execute(ctx context.Context, input CreateCategoryInput) (*entity.Category, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: nama kategori wajib diisi", domainerrors.ErrInvalidInput)
	}
	if !input.CategoryType.IsValid() {
		return nil, fmt.Errorf("%w: category_type harus INCOME, EXPENSE, atau BOTH", domainerrors.ErrInvalidInput)
	}
	icon := strings.TrimSpace(input.Icon)
	if icon == "" {
		icon = "tag"
	}
	color := strings.TrimSpace(input.Color)
	if color == "" {
		color = "#6366f1"
	}
	now := time.Now().UTC()
	cat := &entity.Category{
		ID:           uuid.New(),
		Name:         name,
		Icon:         icon,
		Color:        color,
		CategoryType: input.CategoryType,
		IsDefault:    false,
		IsDeleted:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.catRepo.Create(ctx, input.TenantSchema, cat); err != nil {
		return nil, fmt.Errorf("gagal buat kategori: %w", err)
	}
	return cat, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type UpdateCategoryUseCase struct {
	catRepo repository.CategoryRepository
}

func NewUpdateCategoryUseCase(catRepo repository.CategoryRepository) *UpdateCategoryUseCase {
	return &UpdateCategoryUseCase{catRepo: catRepo}
}

func (uc *UpdateCategoryUseCase) Execute(ctx context.Context, tenantSchema, id, name, icon, color string, catType entity.CategoryType) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: nama kategori wajib diisi", domainerrors.ErrInvalidInput)
	}
	if !catType.IsValid() {
		return fmt.Errorf("%w: category_type harus INCOME, EXPENSE, atau BOTH", domainerrors.ErrInvalidInput)
	}
	existing, err := uc.catRepo.GetByID(ctx, tenantSchema, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domainerrors.ErrCategoryNotFound
	}
	existing.Name = name
	existing.Icon = strings.TrimSpace(icon)
	if existing.Icon == "" {
		existing.Icon = "tag"
	}
	existing.Color = strings.TrimSpace(color)
	if existing.Color == "" {
		existing.Color = "#6366f1"
	}
	existing.CategoryType = catType
	return uc.catRepo.Update(ctx, tenantSchema, existing)
}

// ─────────────────────────────────────────────────────────────────────────────

type DeleteCategoryUseCase struct {
	catRepo repository.CategoryRepository
}

func NewDeleteCategoryUseCase(catRepo repository.CategoryRepository) *DeleteCategoryUseCase {
	return &DeleteCategoryUseCase{catRepo: catRepo}
}

func (uc *DeleteCategoryUseCase) Execute(ctx context.Context, tenantSchema, id string) error {
	cat, err := uc.catRepo.GetByID(ctx, tenantSchema, id)
	if err != nil {
		return err
	}
	if cat == nil {
		return domainerrors.ErrCategoryNotFound
	}
	if cat.IsDefault {
		return domainerrors.ErrDefaultCategoryCannotBeDeleted
	}
	hasActive, err := uc.catRepo.HasActiveTransactions(ctx, tenantSchema, id)
	if err != nil {
		return fmt.Errorf("gagal cek transaksi aktif: %w", err)
	}
	if hasActive {
		return domainerrors.ErrCategoryHasTransactions
	}
	return uc.catRepo.SoftDelete(ctx, tenantSchema, id)
}
