package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/transaction-service/tests/mocks"
)

const reportTenant = "tenant_550e8400_e29b_41d4_a716_446655440000"

func reportDate(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func startOfThisMonth() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// capturedRange menampung rentang yang benar-benar diterima repository. Yang diuji di
// berkas ini memang bukan hasil hitungnya, melainkan PERIODE MANA yang diminta ke DB.
type capturedRange struct{ from, to time.Time }

// Rentang eksplisit dari pemanggil harus diteruskan apa adanya ke tren — termasuk periode
// yang sepenuhnya di masa lalu. Dulu mustahil karena query di-anchor ke NOW(), jadi grafik
// selalu berakhir hari ini walau angka ringkasannya memakai rentang lain.
func TestGetReport_ExplicitRangeDrivesTrend(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockTransactionRepository(ctrl)

	from, to := reportDate(2025, time.January, 1), reportDate(2025, time.March, 31)
	var summary, trend capturedRange

	repo.EXPECT().GetSummary(gomock.Any(), reportTenant, "u1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, f, tt time.Time) (*entity.TransactionSummary, error) {
			summary = capturedRange{f, tt}
			return &entity.TransactionSummary{}, nil
		})
	repo.EXPECT().GetCategoryBreakdown(gomock.Any(), reportTenant, "u1", gomock.Any(), gomock.Any()).
		Return(nil, nil)
	repo.EXPECT().GetMonthlyTrend(gomock.Any(), reportTenant, "u1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, f, tt time.Time) ([]entity.MonthlyPoint, error) {
			trend = capturedRange{f, tt}
			return nil, nil
		})

	uc := usecase.NewGetReportUseCase(repo)
	if _, err := uc.Execute(context.Background(), usecase.GetReportInput{
		TenantSchema: reportTenant, UserID: "u1", From: from, To: to, Months: 6,
	}); err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	if !trend.from.Equal(from) || !trend.to.Equal(to) {
		t.Errorf("tren meminta %s—%s, seharusnya mengikuti rentang eksplisit %s—%s",
			trend.from.Format("2006-01-02"), trend.to.Format("2006-01-02"),
			from.Format("2006-01-02"), to.Format("2006-01-02"))
	}
	// Grafik dan angka ringkasan wajib bicara tentang periode yang sama.
	if !summary.from.Equal(from) || !summary.to.Equal(to) {
		t.Errorf("ringkasan meminta %s—%s, seharusnya %s—%s",
			summary.from.Format("2006-01-02"), summary.to.Format("2006-01-02"),
			from.Format("2006-01-02"), to.Format("2006-01-02"))
	}
}

// Regression: app Android hanya mengirim `months`, tanpa from/to. Jendela tren harus
// tetap mundur N bulan — kalau ikut default ringkasan, grafiknya menyusut jadi satu titik.
func TestGetReport_MonthsOnlyKeepsLegacyTrendWindow(t *testing.T) {
	tests := []struct {
		name       string
		months     int
		wantMonths int
	}{
		{"enam bulan", 6, 6},
		{"dua belas bulan", 12, 12},
		{"nol dianggap default", 0, 6},
		{"di atas batas 24 dianggap default", 25, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockTransactionRepository(ctrl)

			var summary, trend capturedRange
			repo.EXPECT().GetSummary(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _ string, f, to time.Time) (*entity.TransactionSummary, error) {
					summary = capturedRange{f, to}
					return &entity.TransactionSummary{}, nil
				})
			repo.EXPECT().GetCategoryBreakdown(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, nil)
			repo.EXPECT().GetMonthlyTrend(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _ string, f, to time.Time) ([]entity.MonthlyPoint, error) {
					trend = capturedRange{f, to}
					return nil, nil
				})

			uc := usecase.NewGetReportUseCase(repo)
			if _, err := uc.Execute(context.Background(), usecase.GetReportInput{
				TenantSchema: reportTenant, UserID: "u1", Months: tt.months,
			}); err != nil {
				t.Fatalf("tidak mengharapkan error: %v", err)
			}

			wantFrom := startOfThisMonth().AddDate(0, -(tt.wantMonths - 1), 0)
			if !trend.from.Equal(wantFrom) {
				t.Errorf("tren mulai %s, seharusnya %s",
					trend.from.Format("2006-01-02"), wantFrom.Format("2006-01-02"))
			}
			// Default ringkasan (bulan berjalan) tidak boleh ikut berubah.
			if !summary.from.Equal(startOfThisMonth()) {
				t.Errorf("ringkasan mulai %s, seharusnya bulan berjalan %s",
					summary.from.Format("2006-01-02"), startOfThisMonth().Format("2006-01-02"))
			}
		})
	}
}

// Rentang terbalik ditolak, bukan didiamkan: tanpa ini semua angka pulang nol dan
// terbaca user sebagai "tidak punya transaksi", padahal inputnya yang salah.
func TestGetReport_RejectsInvertedRange(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockTransactionRepository(ctrl)
	// Tanpa EXPECT apa pun: gomock otomatis menggagalkan test bila repository disentuh.

	uc := usecase.NewGetReportUseCase(repo)
	_, err := uc.Execute(context.Background(), usecase.GetReportInput{
		TenantSchema: reportTenant, UserID: "u1",
		From: reportDate(2025, time.March, 31), To: reportDate(2025, time.January, 1),
	})
	if !errors.Is(err, domainerrors.ErrInvalidInput) {
		t.Fatalf("error = %v, seharusnya ErrInvalidInput", err)
	}
}

func TestGetReport_RejectsMissingTenantOrUser(t *testing.T) {
	tests := []struct {
		name           string
		tenant, userID string
	}{
		{"tenant kosong", "", "u1"},
		{"user kosong", reportTenant, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockTransactionRepository(ctrl)

			uc := usecase.NewGetReportUseCase(repo)
			_, err := uc.Execute(context.Background(), usecase.GetReportInput{
				TenantSchema: tt.tenant, UserID: tt.userID,
			})
			if !errors.Is(err, domainerrors.ErrInvalidInput) {
				t.Fatalf("error = %v, seharusnya ErrInvalidInput", err)
			}
		})
	}
}
