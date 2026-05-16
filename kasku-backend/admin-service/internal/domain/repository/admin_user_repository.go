package repository

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/google/uuid"
)

// AdminUserRepository adalah port untuk admin_users di kasku_admin.
type AdminUserRepository interface {
	FindByUsername(ctx context.Context, username string) (*entity.AdminUser, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.AdminUser, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error
	Count(ctx context.Context) (int64, error)
	CreateBootstrap(ctx context.Context, admin *entity.AdminUser) error
}
