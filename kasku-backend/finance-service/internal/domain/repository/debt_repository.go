package repository

import (
	"context"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/domain/entity"
)

// DebtRepository mendefinisikan port untuk akses data hutang/piutang.
// SEMUA method menerima tenantSchema untuk multi-tenancy.
type DebtRepository interface {
	Create(ctx context.Context, tenantSchema string, debt *entity.Debt) error
	List(ctx context.Context, tenantSchema, userID string, direction *entity.DebtDirection) ([]entity.Debt, error)
	GetByID(ctx context.Context, tenantSchema, id, userID string) (*entity.Debt, error)
	Update(ctx context.Context, tenantSchema string, debt *entity.Debt) error
	Delete(ctx context.Context, tenantSchema, id, userID string) error

	// Pembayaran
	// RecordPayment mencatat pembayaran dan mengurangi remaining_amount secara ATOMIK
	// dalam satu transaksi (SELECT ... FOR UPDATE) untuk mencegah race condition
	// overpay pada pembayaran konkuren. Mengembalikan ErrDebtNotFound jika hutang
	// tidak ditemukan, ErrDebtAlreadySettled jika sudah lunas, atau ErrPaymentExceedsDebt
	// jika amount melebihi sisa hutang.
	RecordPayment(ctx context.Context, tenantSchema string, debtID, userID string, payment *entity.DebtPayment) error
	ListPayments(ctx context.Context, tenantSchema, debtID string) ([]entity.DebtPayment, error)
}
