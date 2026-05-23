package usecase

import (
	"context"
	"fmt"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
	"github.com/google/uuid"
)

type UpdateBudgetInput struct {
	TenantSchema      string
	UserID            uuid.UUID
	ID                uuid.UUID
	Name              string
	LimitIDR          int64
	CategoryID        *uuid.UUID
	AlertThreshold    int
	DailyLimitEnabled bool
}

type UpdateBudgetUseCase struct {
	repo repository.BudgetRepository
}

func NewUpdateBudgetUseCase(repo repository.BudgetRepository) *UpdateBudgetUseCase {
	return &UpdateBudgetUseCase{repo: repo}
}

func (uc *UpdateBudgetUseCase) Execute(ctx context.Context, input UpdateBudgetInput) error {
	if input.Name == "" {
		return fmt.Errorf("%w: nama anggaran tidak boleh kosong", domainerrors.ErrInvalidInput)
	}
	if input.LimitIDR <= 0 {
		return fmt.Errorf("%w: batas anggaran harus lebih dari 0", domainerrors.ErrInvalidInput)
	}
	if input.AlertThreshold < 0 || input.AlertThreshold > 100 {
		return fmt.Errorf("%w: alert threshold harus antara 0 dan 100", domainerrors.ErrInvalidInput)
	}

	existing, err := uc.repo.GetByID(ctx, input.TenantSchema, input.ID.String(), input.UserID.String())
	if err != nil {
		return err
	}

	if input.DailyLimitEnabled && existing.PeriodType == entity.PeriodCustom {
		return fmt.Errorf("%w: jatah harian hanya tersedia untuk periode MONTHLY atau WEEKLY", domainerrors.ErrInvalidInput)
	}

	existing.Name = input.Name
	existing.LimitIDR = input.LimitIDR
	existing.CategoryID = input.CategoryID
	existing.AlertThreshold = input.AlertThreshold
	existing.DailyLimitEnabled = input.DailyLimitEnabled

	if err := uc.repo.Update(ctx, input.TenantSchema, &existing.Budget); err != nil {
		return fmt.Errorf("gagal update anggaran: %w", err)
	}
	return nil
}
