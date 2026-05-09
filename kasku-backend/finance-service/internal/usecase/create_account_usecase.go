package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/finance-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/repository"
	"github.com/google/uuid"
)

const defaultCurrency = "IDR"

type CreateAccountInput struct {
	TenantSchema string
	UserID       string
	Name         string
	AccountType  entity.AccountType
	Balance      int64
	Currency     string
	Color        string
	Icon         string
	IsDefault    bool
	MaxAccounts  int // dari X-Tier-Max-Accounts header; -1 = unlimited
}

type CreateAccountUseCase struct {
	repo repository.FinancialAccountRepository
}

func NewCreateAccountUseCase(repo repository.FinancialAccountRepository) *CreateAccountUseCase {
	return &CreateAccountUseCase{repo: repo}
}

func (uc *CreateAccountUseCase) Execute(ctx context.Context, input CreateAccountInput) (*entity.FinancialAccount, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: nama akun wajib diisi", domainerrors.ErrInvalidInput)
	}

	// Cek tier limit hanya jika tidak unlimited (MaxAccounts >= 0)
	if input.MaxAccounts >= 0 {
		count, err := uc.repo.CountByUserID(ctx, input.TenantSchema, input.UserID)
		if err != nil {
			return nil, fmt.Errorf("gagal hitung akun: %w", err)
		}
		if count >= input.MaxAccounts {
			return nil, domainerrors.ErrAccountLimitReached
		}
	}

	if input.Currency == "" {
		input.Currency = defaultCurrency
	}

	now := time.Now().UTC()
	account := &entity.FinancialAccount{
		ID:          uuid.New(),
		UserID:      uuid.MustParse(input.UserID),
		Name:        input.Name,
		AccountType: input.AccountType,
		Balance:     input.Balance,
		Currency:    input.Currency,
		Color:       input.Color,
		Icon:        input.Icon,
		IsDefault:   input.IsDefault,
		IsDeleted:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.repo.Create(ctx, input.TenantSchema, account); err != nil {
		return nil, fmt.Errorf("gagal buat akun: %w", err)
	}

	return account, nil
}
