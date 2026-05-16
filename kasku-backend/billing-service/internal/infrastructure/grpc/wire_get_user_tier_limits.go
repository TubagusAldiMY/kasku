package grpc

import (
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"google.golang.org/protobuf/encoding/protowire"
)

// decodeGetUserTierLimitsRequest mendecode request proto3 wire-format:
//
//	field 1 (BytesType): user_id (string)
//
// Field number & wire type WAJIB konsisten dengan api-gateway/proto/billing/v1/billing_grpc.go
// (encodeRequest). Lihat juga billing-service/proto/billing/v1/billing.proto.
func decodeGetUserTierLimitsRequest(b []byte) (string, error) {
	var userID string
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return "", protowire.ParseError(n)
		}
		b = b[n:]

		switch {
		case num == 1 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return "", protowire.ParseError(n)
			}
			userID = s
			b = b[n:]
		default:
			// Skip field yang tidak dikenal agar forward-compatible.
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return "", protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return userID, nil
}

// encodeTierLimitsResponse mengkode response ke proto3 wire-format:
//
//	field 1 (varint): max_transactions_per_month (int32)
//	field 2 (varint): max_financial_accounts (int32)
//	field 3 (varint): max_investment_instruments (int32)
//	field 4 (varint): history_retention_months (int32)
//	field 5 (varint): email_notifications_enabled (bool)
//	field 6 (varint): export_csv_enabled (bool)
//
// Urutan field & nomor HARUS sinkron dengan decodeResponse di
// api-gateway/proto/billing/v1/billing_grpc.go.
func encodeTierLimitsResponse(limits *entity.PlanLimits) []byte {
	var b []byte

	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(limits.MaxTransactionsPerMonth))

	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(limits.MaxFinancialAccounts))

	b = protowire.AppendTag(b, 3, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(limits.MaxInvestmentInstruments))

	b = protowire.AppendTag(b, 4, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(limits.HistoryRetentionMonths))

	var emailEnabled uint64
	if limits.EmailNotificationsEnabled {
		emailEnabled = 1
	}
	b = protowire.AppendTag(b, 5, protowire.VarintType)
	b = protowire.AppendVarint(b, emailEnabled)

	var csvEnabled uint64
	if limits.ExportCsvEnabled {
		csvEnabled = 1
	}
	b = protowire.AppendTag(b, 6, protowire.VarintType)
	b = protowire.AppendVarint(b, csvEnabled)

	return b
}
