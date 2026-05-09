package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"

	domainerrors "github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/domain/repository"
)

type ExportCSVUseCase struct {
	txRepo repository.TransactionRepository
}

func NewExportCSVUseCase(txRepo repository.TransactionRepository) *ExportCSVUseCase {
	return &ExportCSVUseCase{txRepo: txRepo}
}

func (uc *ExportCSVUseCase) Execute(ctx context.Context, tenantSchema, userID string, exportAllowed bool) ([]byte, error) {
	if !exportAllowed {
		return nil, domainerrors.ErrExportNotAllowed
	}

	txs, err := uc.txRepo.ListForExport(ctx, tenantSchema, userID, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil data untuk export: %w", err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write([]string{
		"ID", "Sync ID", "Account ID", "Category ID", "Type",
		"Amount (IDR)", "Date", "Notes", "To Account ID", "Created At",
	}); err != nil {
		return nil, fmt.Errorf("gagal tulis CSV header: %w", err)
	}

	for _, tx := range txs {
		categoryID := ""
		if tx.CategoryID != nil {
			categoryID = tx.CategoryID.String()
		}
		toAccountID := ""
		if tx.ToAccountID != nil {
			toAccountID = tx.ToAccountID.String()
		}
		if err := w.Write([]string{
			tx.ID.String(),
			tx.SyncID,
			tx.AccountID.String(),
			categoryID,
			string(tx.TransactionType),
			strconv.FormatInt(tx.AmountIDR, 10),
			tx.TransactionDate.Format("2006-01-02"),
			tx.Notes,
			toAccountID,
			tx.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}); err != nil {
			return nil, fmt.Errorf("gagal tulis CSV row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("gagal flush CSV writer: %w", err)
	}

	return buf.Bytes(), nil
}
