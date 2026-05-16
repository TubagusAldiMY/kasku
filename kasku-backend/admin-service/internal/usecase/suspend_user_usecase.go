package usecase

import (
	"context"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
)

// SuspendUserInput dikirim oleh handler.
type SuspendUserInput struct {
	AdminID      uuid.UUID
	TargetUserID uuid.UUID
	Reason       string
}

// SuspendUserUseCase menonaktifkan user di kasku_auth + audit log.
type SuspendUserUseCase interface {
	Execute(ctx context.Context, in SuspendUserInput) error
}

type suspendUserUseCase struct {
	userRead  repository.UserReadRepository
	userWrite repository.UserWriteRepository
	audit     *AuditLogger
}

// NewSuspendUserUseCase membuat instance.
func NewSuspendUserUseCase(userRead repository.UserReadRepository, userWrite repository.UserWriteRepository, audit *AuditLogger) SuspendUserUseCase {
	return &suspendUserUseCase{userRead: userRead, userWrite: userWrite, audit: audit}
}

func (uc *suspendUserUseCase) Execute(ctx context.Context, in SuspendUserInput) error {
	// Pre-check: user harus ada
	user, err := uc.userRead.GetByID(ctx, in.TargetUserID)
	if err != nil {
		return err
	}
	if user == nil {
		return domainerrors.ErrUserNotFound
	}

	mutationErr := uc.userWrite.SetIsActive(ctx, in.TargetUserID, false)
	targetUserID := in.TargetUserID
	uc.audit.Log(ctx, AuditInput{
		AdminID:      in.AdminID,
		Action:       entity.AuditActionSuspendUser,
		TargetUserID: &targetUserID,
		TargetEntity: strPtr("user"),
		Metadata: map[string]any{
			"reason":  in.Reason,
			"email":   user.Email,
		},
		Success: mutationErr == nil,
	})
	return mutationErr
}

func strPtr(s string) *string { return &s }
