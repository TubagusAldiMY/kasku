package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
	"github.com/google/uuid"
)

type CreateBudgetInput struct {
	TenantSchema      string
	UserID            uuid.UUID
	SyncID            string
	Name              string
	LimitIDR          int64
	CategoryID        *uuid.UUID
	PeriodType        entity.BudgetPeriodType
	StartDate         time.Time
	EndDate           *time.Time
	AlertThreshold    int
	DailyLimitEnabled bool
	MaxBudgets        int // -1 = unlimited
}

type CreateBudgetUseCase struct {
	repo repository.BudgetRepository
}

func NewCreateBudgetUseCase(repo repository.BudgetRepository) *CreateBudgetUseCase {
	return &CreateBudgetUseCase{repo: repo}
}

func (uc *CreateBudgetUseCase) Execute(ctx context.Context, input CreateBudgetInput) (*entity.Budget, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: nama anggaran tidak boleh kosong", domainerrors.ErrInvalidInput)
	}
	if input.LimitIDR <= 0 {
		return nil, fmt.Errorf("%w: batas anggaran harus lebih dari 0", domainerrors.ErrInvalidInput)
	}
	if input.AlertThreshold < 0 || input.AlertThreshold > 100 {
		return nil, fmt.Errorf("%w: alert threshold harus antara 0 dan 100", domainerrors.ErrInvalidInput)
	}

	periodType := input.PeriodType
	if periodType == "" {
		periodType = entity.PeriodMonthly
	}

	if input.DailyLimitEnabled && periodType == entity.PeriodCustom {
		return nil, fmt.Errorf("%w: jatah harian hanya tersedia untuk periode MONTHLY atau WEEKLY", domainerrors.ErrInvalidInput)
	}

	if input.MaxBudgets != -1 {
		count, err := uc.repo.Count(ctx, input.TenantSchema, input.UserID.String())
		if err != nil {
			return nil, fmt.Errorf("gagal cek jumlah anggaran: %w", err)
		}
		if count >= input.MaxBudgets {
			return nil, domainerrors.ErrBudgetLimitReached
		}
	}

	alertThreshold := input.AlertThreshold
	if alertThreshold == 0 {
		alertThreshold = 80
	}

	startDate := input.StartDate
	if startDate.IsZero() {
		startDate = time.Now().UTC()
	}

	syncID := input.SyncID
	if syncID == "" {
		syncID = uuid.New().String()
	}

	now := time.Now().UTC()
	budget := &entity.Budget{
		ID:                uuid.New(),
		UserID:            input.UserID,
		SyncID:            syncID,
		Name:              input.Name,
		LimitIDR:          input.LimitIDR,
		CategoryID:        input.CategoryID,
		PeriodType:        periodType,
		StartDate:         startDate,
		EndDate:           input.EndDate,
		AlertThreshold:    alertThreshold,
		DailyLimitEnabled: input.DailyLimitEnabled,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := uc.repo.Create(ctx, input.TenantSchema, budget); err != nil {
		return nil, fmt.Errorf("gagal buat anggaran: %w", err)
	}
	return budget, nil
}
