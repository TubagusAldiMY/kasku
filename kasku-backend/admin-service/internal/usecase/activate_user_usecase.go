package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
)

// ActivateUserInput dikirim oleh handler.
type ActivateUserInput struct {
	AdminID      uuid.UUID
	TargetUserID uuid.UUID
	Reason       string
}

// ActivateUserUseCase mengaktifkan kembali user yang sebelumnya disuspend.
type ActivateUserUseCase interface {
	Execute(ctx context.Context, in ActivateUserInput) error
}

type activateUserUseCase struct {
	userRead  repository.UserReadRepository
	userWrite repository.UserWriteRepository
	audit     *AuditLogger
}

// NewActivateUserUseCase membuat instance.
func NewActivateUserUseCase(userRead repository.UserReadRepository, userWrite repository.UserWriteRepository, audit *AuditLogger) ActivateUserUseCase {
	return &activateUserUseCase{userRead: userRead, userWrite: userWrite, audit: audit}
}

func (uc *activateUserUseCase) Execute(ctx context.Context, in ActivateUserInput) error {
	user, err := uc.userRead.GetByID(ctx, in.TargetUserID)
	if err != nil {
		return err
	}
	if user == nil {
		return domainerrors.ErrUserNotFound
	}

	mutationErr := uc.userWrite.SetIsActive(ctx, in.TargetUserID, true)
	targetUserID := in.TargetUserID
	uc.audit.Log(ctx, AuditInput{
		AdminID:      in.AdminID,
		Action:       entity.AuditActionActivateUser,
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
