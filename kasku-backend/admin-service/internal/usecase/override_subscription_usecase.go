package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/google/uuid"
)

// OverrideSubscriptionInput dikirim handler.
type OverrideSubscriptionInput struct {
	AdminID      uuid.UUID
	TargetUserID uuid.UUID
	NewPlanName  string // "FREE" | "BASIC" | "PRO"
	Reason       string
}

// OverrideSubscriptionUseCase mengganti tier subscription user tanpa pembayaran (admin override).
type OverrideSubscriptionUseCase interface {
	Execute(ctx context.Context, in OverrideSubscriptionInput) error
}

type overrideSubscriptionUseCase struct {
	subRepo repository.SubscriptionRepository
	audit   *AuditLogger
}

// NewOverrideSubscriptionUseCase membuat instance.
func NewOverrideSubscriptionUseCase(subRepo repository.SubscriptionRepository, audit *AuditLogger) OverrideSubscriptionUseCase {
	return &overrideSubscriptionUseCase{subRepo: subRepo, audit: audit}
}

func (uc *overrideSubscriptionUseCase) Execute(ctx context.Context, in OverrideSubscriptionInput) error {
	if strings.TrimSpace(in.Reason) == "" {
		return domainerrors.ErrValidation
	}

	currentSub, err := uc.subRepo.GetByUserID(ctx, in.TargetUserID)
	if err != nil {
		return err
	}
	if currentSub == nil {
		return domainerrors.ErrSubscriptionNotFound
	}

	newPlanID, newPrice, err := uc.subRepo.FindPlanByName(ctx, in.NewPlanName)
	if err != nil {
		return err
	}
	if newPlanID == uuid.Nil {
		return domainerrors.ErrPlanNotFound
	}

	mutationErr := uc.subRepo.UpdatePlan(ctx, currentSub.ID, newPlanID, time.Now().UTC())

	targetUserID := in.TargetUserID
	uc.audit.Log(ctx, AuditInput{
		AdminID:      in.AdminID,
		Action:       entity.AuditActionOverrideSubscription,
		TargetUserID: &targetUserID,
		TargetEntity: strPtr("subscription"),
		Metadata: map[string]any{
			"subscription_id": currentSub.ID.String(),
			"old_plan":        currentSub.PlanName,
			"new_plan":        strings.ToUpper(in.NewPlanName),
			"new_price_idr":   newPrice,
			"reason":          in.Reason,
		},
		Success: mutationErr == nil,
	})
	return mutationErr
}
