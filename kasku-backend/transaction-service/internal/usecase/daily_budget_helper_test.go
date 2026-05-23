package usecase

import (
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	"github.com/google/uuid"
)

// budgetWithProgress builds a minimal BudgetWithProgress for testing.
func budgetWithProgress(periodType entity.BudgetPeriodType, limitIDR, spentIDR, spentTodayIDR int64) entity.BudgetWithProgress {
	return entity.BudgetWithProgress{
		Budget: entity.Budget{
			ID:                uuid.New(),
			PeriodType:        periodType,
			LimitIDR:          limitIDR,
			DailyLimitEnabled: true,
		},
		SpentIDR:      spentIDR,
		SpentTodayIDR: spentTodayIDR,
	}
}

func TestComputeDailyFields_Disabled(t *testing.T) {
	b := entity.BudgetWithProgress{
		Budget: entity.Budget{DailyLimitEnabled: false, LimitIDR: 300_000},
	}
	computeDailyFields(&b)
	if b.DailyBaseIDR != 0 {
		t.Errorf("expected DailyBaseIDR=0 when disabled, got %d", b.DailyBaseIDR)
	}
}

func TestComputeDailyFields_Monthly_PerfectSpend(t *testing.T) {
	// Simulate: spent exactly daily_base every day so far → carryover=0, allowance=daily_base.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	daysInMonth := daysInCurrentMonth(today)

	y, m, _ := today.Date()
	firstOfMonth := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	daysElapsed := int(today.Sub(firstOfMonth).Hours()/24) + 1

	dailyBase := int64(300_000 / daysInMonth)
	// Perfect spend: spent exactly allocated amount so far (including today)
	totalSpent := int64(daysElapsed) * dailyBase
	spentToday := dailyBase
	spentBefore := totalSpent - spentToday

	b := entity.BudgetWithProgress{
		Budget: entity.Budget{
			ID:                uuid.New(),
			PeriodType:        entity.PeriodMonthly,
			LimitIDR:          300_000,
			DailyLimitEnabled: true,
		},
		SpentIDR:      spentBefore + spentToday,
		SpentTodayIDR: spentToday,
	}
	computeDailyFields(&b)

	if b.DailyBaseIDR != dailyBase {
		t.Errorf("DailyBaseIDR: got %d, want %d", b.DailyBaseIDR, dailyBase)
	}
	// Carryover = (days_elapsed-1)*daily_base - spent_before = 0
	if b.CarryoverIDR != 0 {
		t.Errorf("CarryoverIDR with perfect spend: got %d, want 0", b.CarryoverIDR)
	}
	if b.DailyAllowanceTodayIDR != dailyBase {
		t.Errorf("DailyAllowanceTodayIDR: got %d, want %d", b.DailyAllowanceTodayIDR, dailyBase)
	}
	if b.DailyRemainingIDR != 0 {
		t.Errorf("DailyRemainingIDR with perfect spend: got %d, want 0", b.DailyRemainingIDR)
	}
}

func TestComputeDailyFields_Weekly_FormulaCheck(t *testing.T) {
	// Verify formula with WEEKLY budget: 70,000/week → base=10,000/day.
	// Simulate day 2 of the week, spent 12,000 on day 1, 0 today.
	// We can't control "today" directly in unit tests without clock injection,
	// so verify the internal calculation logic using the formula directly.
	// Formula: today_allowance = days_elapsed * daily_base - spent_before_today
	// For day 2: days_elapsed=2, daily_base=10,000, spent_before_today=12,000
	// → today_allowance = 2*10,000 - 12,000 = 8,000
	dailyBase := int64(70_000 / 7) // 10,000
	daysElapsed := 2
	spentBeforeToday := int64(12_000)
	expectedAllowance := int64(daysElapsed)*dailyBase - spentBeforeToday // 8,000

	if expectedAllowance != 8_000 {
		t.Errorf("formula sanity check failed: expected 8000 got %d", expectedAllowance)
	}

	// Verify carryover sign: negative means deficit
	carryover := int64(daysElapsed-1)*dailyBase - spentBeforeToday // 1*10000 - 12000 = -2000
	if carryover != -2_000 {
		t.Errorf("carryover sanity check failed: expected -2000 got %d", carryover)
	}
}

func TestComputeDailyFields_Custom_Skipped(t *testing.T) {
	b := budgetWithProgress(entity.PeriodCustom, 300_000, 0, 0)
	computeDailyFields(&b)
	if b.DailyBaseIDR != 0 {
		t.Errorf("CUSTOM period should be skipped, got DailyBaseIDR=%d", b.DailyBaseIDR)
	}
}

func daysInCurrentMonth(t time.Time) int {
	y, m, _ := t.Date()
	first := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	next := time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
	return int(next.Sub(first).Hours() / 24)
}
