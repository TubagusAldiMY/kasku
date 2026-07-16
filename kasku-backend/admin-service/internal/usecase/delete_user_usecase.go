package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
)

// DeleteUserInput dikirim oleh handler.
type DeleteUserInput struct {
	AdminID      uuid.UUID
	TargetUserID uuid.UUID
	Reason       string
}

// DeleteUserUseCase melakukan soft-delete + anonymisasi user di kasku_auth + audit log.
// Data tenant/billing tidak dihapus (retensi untuk keperluan akunting/legal).
type DeleteUserUseCase interface {
	Execute(ctx context.Context, in DeleteUserInput) error
}

type deleteUserUseCase struct {
	userRead  repository.UserReadRepository
	userWrite repository.UserWriteRepository
	audit     *AuditLogger
}

// NewDeleteUserUseCase membuat instance.
func NewDeleteUserUseCase(userRead repository.UserReadRepository, userWrite repository.UserWriteRepository, audit *AuditLogger) DeleteUserUseCase {
	return &deleteUserUseCase{userRead: userRead, userWrite: userWrite, audit: audit}
}

func (uc *deleteUserUseCase) Execute(ctx context.Context, in DeleteUserInput) error {
	// Pre-check: user harus ada (ambil email asli untuk audit sebelum dianonimkan).
	user, err := uc.userRead.GetByID(ctx, in.TargetUserID)
	if err != nil {
		return err
	}
	if user == nil {
		return domainerrors.ErrUserNotFound
	}

	mutationErr := uc.userWrite.SoftDeleteAndAnonymize(ctx, in.TargetUserID)
	targetUserID := in.TargetUserID
	uc.audit.Log(ctx, AuditInput{
		AdminID:      in.AdminID,
		Action:       entity.AuditActionDeleteUser,
		TargetUserID: &targetUserID,
		TargetEntity: strPtr("user"),
		Metadata: map[string]any{
			"reason": in.Reason,
			"email":  user.Email,
		},
		Success: mutationErr == nil,
	})
	return mutationErr
}
