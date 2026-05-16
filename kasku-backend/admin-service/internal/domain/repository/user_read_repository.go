package repository

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/google/uuid"
)

// UserListFilter adalah opsi filter untuk list users di admin dashboard.
type UserListFilter struct {
	Query         string // LIKE email/username
	IsActive      *bool
	EmailVerified *bool
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	Limit         int
	Offset        int
}

// UserReadRepository adalah port untuk membaca user dari kasku_auth.
type UserReadRepository interface {
	List(ctx context.Context, filter UserListFilter) ([]entity.UserSummary, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.UserSummary, error)
	// CountTotal dipakai dashboard stats.
	CountTotal(ctx context.Context) (int64, error)
	CountActive(ctx context.Context) (int64, error)
	CountCreatedSince(ctx context.Context, since time.Time) (int64, error)
}

// UserWriteRepository adalah port untuk memperbarui flag pada user (kasku_auth).
// Saat ini hanya mendukung toggle is_active (suspend/activate).
type UserWriteRepository interface {
	SetIsActive(ctx context.Context, userID uuid.UUID, isActive bool) error
}
