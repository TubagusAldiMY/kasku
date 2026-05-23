package usecase

import (
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
)

// computeDailyFields mengisi field jatah harian pada BudgetWithProgress.
// Formula:
//   daily_base        = limit / days_in_period
//   days_elapsed      = (today - period_start) + 1
//   carryover         = (days_elapsed-1) * daily_base - spent_before_today
//   today_allowance   = daily_base + carryover
//   remaining_today   = today_allowance - spent_today
func computeDailyFields(b *entity.BudgetWithProgress) {
	if !b.DailyLimitEnabled {
		return
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	var daysInPeriod int
	var periodStart time.Time

	switch b.PeriodType {
	case entity.PeriodMonthly:
		y, m, _ := today.Date()
		first := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
		next := time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
		daysInPeriod = int(next.Sub(first).Hours() / 24)
		periodStart = first
	case entity.PeriodWeekly:
		daysInPeriod = 7
		// ISO week: Monday = day 1
		wd := int(today.Weekday())
		if wd == 0 {
			wd = 7 // Sunday is 7 in ISO
		}
		periodStart = today.AddDate(0, 0, -(wd - 1))
	default:
		// CUSTOM tidak support daily tracking
		return
	}

	if daysInPeriod == 0 {
		return
	}

	daysElapsed := max(int(today.Sub(periodStart).Hours()/24)+1, 1)

	b.DailyBaseIDR = b.LimitIDR / int64(daysInPeriod)
	spentBeforeToday := b.SpentIDR - b.SpentTodayIDR
	b.CarryoverIDR = int64(daysElapsed-1)*b.DailyBaseIDR - spentBeforeToday
	b.DailyAllowanceTodayIDR = b.DailyBaseIDR + b.CarryoverIDR
	b.DailyRemainingIDR = b.DailyAllowanceTodayIDR - b.SpentTodayIDR
}
