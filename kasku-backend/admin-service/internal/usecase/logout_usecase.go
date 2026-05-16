package usecase

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/redis"
	"github.com/google/uuid"
)

// LogoutInput berisi data yang diperlukan untuk blacklist token.
type LogoutInput struct {
	AdminID   uuid.UUID
	JTI       string
	ExpiresAt time.Time
}

// LogoutUseCase mengelola alur logout admin (blacklist JTI).
type LogoutUseCase interface {
	Execute(ctx context.Context, in LogoutInput) error
}

type logoutUseCase struct {
	blacklist *redis.TokenBlacklist
	audit     *AuditLogger
}

// NewLogoutUseCase membuat instance.
func NewLogoutUseCase(blacklist *redis.TokenBlacklist, audit *AuditLogger) LogoutUseCase {
	return &logoutUseCase{blacklist: blacklist, audit: audit}
}

// Execute mem-blacklist JTI sampai expiry asli dan mencatat LOGOUT.
func (uc *logoutUseCase) Execute(ctx context.Context, in LogoutInput) error {
	remaining := time.Until(in.ExpiresAt)
	if err := uc.blacklist.Blacklist(ctx, in.JTI, remaining); err != nil {
		return err
	}
	uc.audit.Log(ctx, AuditInput{
		AdminID: in.AdminID,
		Action:  entity.AuditActionLogout,
		Metadata: map[string]any{
			"jti": in.JTI,
		},
		Success: true,
	})
	return nil
}
