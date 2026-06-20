package usecase

import (
	"context"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
)

type GetReportInput struct {
	TenantSchema string
	UserID       string
	From         time.Time
	To           time.Time
	Months       int
}

type GetReportUseCase struct {
	txRepo repository.TransactionRepository
}

func NewGetReportUseCase(txRepo repository.TransactionRepository) *GetReportUseCase {
	return &GetReportUseCase{txRepo: txRepo}
}

func (uc *GetReportUseCase) Execute(ctx context.Context, input GetReportInput) (*entity.ReportData, error) {
	if input.TenantSchema == "" || input.UserID == "" {
		return nil, domainerrors.ErrInvalidInput
	}
	if input.Months <= 0 || input.Months > 24 {
		input.Months = 6
	}

	now := time.Now().UTC()
	if input.From.IsZero() {
		input.From = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if input.To.IsZero() {
		input.To = now
	}

	summary, err := uc.txRepo.GetSummary(ctx, input.TenantSchema, input.UserID, input.From, input.To)
	if err != nil {
		return nil, err
	}
	breakdown, err := uc.txRepo.GetCategoryBreakdown(ctx, input.TenantSchema, input.UserID, input.From, input.To)
	if err != nil {
		return nil, err
	}
	trend, err := uc.txRepo.GetMonthlyTrend(ctx, input.TenantSchema, input.UserID, input.Months)
	if err != nil {
		return nil, err
	}

	return &entity.ReportData{
		PeriodFrom:        input.From.Format("2006-01-02"),
		PeriodTo:          input.To.Format("2006-01-02"),
		TotalIncome:       summary.TotalIncome,
		TotalExpense:      summary.TotalExpense,
		NetAmount:         summary.NetAmount,
		CategoryBreakdown: breakdown,
		MonthlyTrend:      trend,
	}, nil
}
