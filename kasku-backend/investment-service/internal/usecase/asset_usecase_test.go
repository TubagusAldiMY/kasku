package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/investment-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/usecase"
	"github.com/google/uuid"
)

// fakeRepo adalah hand-rolled fake (bukan generated mock) untuk menghindari
// dependency tambahan ke gomock. Cukup untuk verifikasi behavior usecase.
type fakeRepo struct {
	countActiveValue int
	countActiveErr   error

	createCalledWith *entity.InvestmentAsset
	createErr        error

	getByIDValue *entity.InvestmentAsset
	getByIDErr   error

	createHistoryCalled int
	createHistoryErr    error
	lastHistoryEntry    *entity.UnitHistory

	updateQuantityCalled       int
	updateQuantityLastQty      float64
	updateQuantityLastAvgPrice float64
	updateQuantityErr          error
}

func (f *fakeRepo) CountActive(_ context.Context, _ string) (int, error) {
	return f.countActiveValue, f.countActiveErr
}

func (f *fakeRepo) Create(_ context.Context, _ string, asset *entity.InvestmentAsset) error {
	f.createCalledWith = asset
	return f.createErr
}

func (f *fakeRepo) List(_ context.Context, _ string) ([]entity.InvestmentAsset, error) {
	return nil, nil
}

func (f *fakeRepo) GetByID(_ context.Context, _, _ string) (*entity.InvestmentAsset, error) {
	return f.getByIDValue, f.getByIDErr
}

func (f *fakeRepo) Update(_ context.Context, _ string, _ *entity.InvestmentAsset) error {
	return nil
}

func (f *fakeRepo) SoftDelete(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeRepo) CreateUnitHistory(_ context.Context, _ string, entry *entity.UnitHistory) error {
	f.createHistoryCalled++
	f.lastHistoryEntry = entry
	return f.createHistoryErr
}

func (f *fakeRepo) GetUnitHistory(_ context.Context, _, _ string, _ int) ([]entity.UnitHistory, error) {
	return nil, nil
}

func (f *fakeRepo) UpdateQuantity(_ context.Context, _, _ string, newQty, newAvg float64) error {
	f.updateQuantityCalled++
	f.updateQuantityLastQty = newQty
	f.updateQuantityLastAvgPrice = newAvg
	return f.updateQuantityErr
}

// ---------------- CreateAssetUseCase ----------------

func TestCreateAssetUseCase_RejectsEmptyName(t *testing.T) {
	t.Parallel()
	uc := usecase.NewCreateAssetUseCase(&fakeRepo{})
	_, err := uc.Execute(context.Background(), usecase.CreateAssetInput{
		TenantSchema:   "tenant_x",
		Name:           "",
		Symbol:         "BTC",
		MaxInvestments: -1,
	})
	if !errors.Is(err, domainerrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateAssetUseCase_RejectsEmptySymbol(t *testing.T) {
	t.Parallel()
	uc := usecase.NewCreateAssetUseCase(&fakeRepo{})
	_, err := uc.Execute(context.Background(), usecase.CreateAssetInput{
		TenantSchema:   "tenant_x",
		Name:           "Bitcoin",
		Symbol:         "",
		MaxInvestments: -1,
	})
	if !errors.Is(err, domainerrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateAssetUseCase_HitsTierLimit(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{countActiveValue: 5}
	uc := usecase.NewCreateAssetUseCase(repo)
	_, err := uc.Execute(context.Background(), usecase.CreateAssetInput{
		TenantSchema:   "tenant_x",
		Name:           "Bitcoin",
		Symbol:         "BTC",
		MaxInvestments: 5,
	})
	if !errors.Is(err, domainerrors.ErrAssetLimitReached) {
		t.Fatalf("expected ErrAssetLimitReached, got %v", err)
	}
}

func TestCreateAssetUseCase_UnlimitedTierSkipsCountCheck(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{countActiveValue: 999, countActiveErr: errors.New("should not be called")}
	uc := usecase.NewCreateAssetUseCase(repo)
	_, err := uc.Execute(context.Background(), usecase.CreateAssetInput{
		TenantSchema:   "tenant_x",
		Name:           "Bitcoin",
		Symbol:         "BTC",
		AssetType:      entity.AssetTypeCrypto,
		Quantity:       0,
		MaxInvestments: -1, // unlimited
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.createCalledWith == nil {
		t.Fatalf("expected Create to be called")
	}
	if repo.createHistoryCalled != 0 {
		t.Fatalf("expected 0 history entries for quantity=0, got %d", repo.createHistoryCalled)
	}
}

func TestCreateAssetUseCase_RecordsInitialHistoryWhenQuantityPositive(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	uc := usecase.NewCreateAssetUseCase(repo)
	_, err := uc.Execute(context.Background(), usecase.CreateAssetInput{
		TenantSchema:   "tenant_x",
		Name:           "Bitcoin",
		Symbol:         "BTC",
		AssetType:      entity.AssetTypeCrypto,
		Quantity:       0.5,
		AvgBuyPrice:    100_000_000,
		MaxInvestments: -1,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.createHistoryCalled != 1 {
		t.Fatalf("expected 1 history entry, got %d", repo.createHistoryCalled)
	}
	if repo.lastHistoryEntry.TransactionType != "BUY" {
		t.Fatalf("expected BUY, got %s", repo.lastHistoryEntry.TransactionType)
	}
	if repo.lastHistoryEntry.QuantityChange != 0.5 {
		t.Fatalf("expected qty 0.5, got %f", repo.lastHistoryEntry.QuantityChange)
	}
}

func TestCreateAssetUseCase_DefaultsCurrencyToIDR(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	uc := usecase.NewCreateAssetUseCase(repo)
	_, err := uc.Execute(context.Background(), usecase.CreateAssetInput{
		TenantSchema:   "tenant_x",
		Name:           "Emas",
		Symbol:         "XAU",
		AssetType:      entity.AssetTypeGold,
		Currency:       "",
		MaxInvestments: -1,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.createCalledWith.Currency != "IDR" {
		t.Fatalf("expected default currency IDR, got %s", repo.createCalledWith.Currency)
	}
}

// ---------------- RecordUnitChangeUseCase ----------------

func TestRecordUnitChange_RejectsInvalidTxType(t *testing.T) {
	t.Parallel()
	uc := usecase.NewRecordUnitChangeUseCase(&fakeRepo{})
	_, err := uc.Execute(context.Background(), usecase.RecordUnitChangeInput{
		TenantSchema:    "tenant_x",
		AssetID:         uuid.NewString(),
		TransactionType: "INVALID",
	})
	if !errors.Is(err, domainerrors.ErrInvalidTransactionType) {
		t.Fatalf("expected ErrInvalidTransactionType, got %v", err)
	}
}

func TestRecordUnitChange_BuyRecomputesAvgBuyPrice(t *testing.T) {
	t.Parallel()
	// Existing: 1.0 BTC @ 100M IDR  → total value 100M
	// BUY     : 1.0 BTC @ 200M IDR  → adds 200M
	// New qty : 2.0 → new avg = 300M / 2 = 150M
	existingAsset := &entity.InvestmentAsset{
		ID:          uuid.New(),
		Quantity:    1.0,
		AvgBuyPrice: 100_000_000,
	}
	repo := &fakeRepo{getByIDValue: existingAsset}
	uc := usecase.NewRecordUnitChangeUseCase(repo)
	_, err := uc.Execute(context.Background(), usecase.RecordUnitChangeInput{
		TenantSchema:    "tenant_x",
		AssetID:         existingAsset.ID.String(),
		TransactionType: "BUY",
		QuantityChange:  1.0,
		PricePerUnit:    200_000_000,
		TransactionDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.updateQuantityLastQty != 2.0 {
		t.Fatalf("expected new qty 2.0, got %f", repo.updateQuantityLastQty)
	}
	if repo.updateQuantityLastAvgPrice != 150_000_000 {
		t.Fatalf("expected avg 150M, got %f", repo.updateQuantityLastAvgPrice)
	}
}

func TestRecordUnitChange_SellDoesNotChangeAvgBuyPrice(t *testing.T) {
	t.Parallel()
	existingAsset := &entity.InvestmentAsset{
		ID:          uuid.New(),
		Quantity:    2.0,
		AvgBuyPrice: 150_000_000,
	}
	repo := &fakeRepo{getByIDValue: existingAsset}
	uc := usecase.NewRecordUnitChangeUseCase(repo)
	_, err := uc.Execute(context.Background(), usecase.RecordUnitChangeInput{
		TenantSchema:    "tenant_x",
		AssetID:         existingAsset.ID.String(),
		TransactionType: "SELL",
		QuantityChange:  -1.0,
		PricePerUnit:    200_000_000,
		TransactionDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.updateQuantityLastQty != 1.0 {
		t.Fatalf("expected new qty 1.0, got %f", repo.updateQuantityLastQty)
	}
	if repo.updateQuantityLastAvgPrice != 150_000_000 {
		t.Fatalf("expected avg unchanged 150M, got %f", repo.updateQuantityLastAvgPrice)
	}
}

func TestRecordUnitChange_RejectsNegativeResultingQuantity(t *testing.T) {
	t.Parallel()
	existingAsset := &entity.InvestmentAsset{
		ID:       uuid.New(),
		Quantity: 1.0,
	}
	repo := &fakeRepo{getByIDValue: existingAsset}
	uc := usecase.NewRecordUnitChangeUseCase(repo)
	_, err := uc.Execute(context.Background(), usecase.RecordUnitChangeInput{
		TenantSchema:    "tenant_x",
		AssetID:         existingAsset.ID.String(),
		TransactionType: "SELL",
		QuantityChange:  -2.0,
		PricePerUnit:    1,
		TransactionDate: time.Now(),
	})
	if !errors.Is(err, domainerrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for negative qty, got %v", err)
	}
	if repo.updateQuantityCalled != 0 {
		t.Fatalf("expected UpdateQuantity NOT to be called on validation failure")
	}
}
