package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
)

// GetCurrentAdminUseCase mengembalikan profil admin berdasarkan ID dari JWT.
type GetCurrentAdminUseCase interface {
	Execute(ctx context.Context, adminID uuid.UUID) (*entity.AdminUser, error)
}

type getCurrentAdminUseCase struct {
	repo repository.AdminUserRepository
}

// NewGetCurrentAdminUseCase membuat instance.
func NewGetCurrentAdminUseCase(repo repository.AdminUserRepository) GetCurrentAdminUseCase {
	return &getCurrentAdminUseCase{repo: repo}
}

func (uc *getCurrentAdminUseCase) Execute(ctx context.Context, id uuid.UUID) (*entity.AdminUser, error) {
	admin, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, domainerrors.ErrAdminNotFound
	}
	if !admin.IsActive {
		return nil, domainerrors.ErrAdminInactive
	}
	return admin, nil
}
