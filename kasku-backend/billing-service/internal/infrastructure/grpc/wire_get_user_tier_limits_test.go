package grpc

import (
	"testing"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

// TestDecodeGetUserTierLimitsRequest memverifikasi decoder kompatibel dengan
// encoder di api-gateway/proto/billing/v1/billing_grpc.go (field 1 BytesType string).
func TestDecodeGetUserTierLimitsRequest(t *testing.T) {
	t.Run("decodes user_id from field 1", func(t *testing.T) {
		var b []byte
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, "user-uuid-123")

		got, err := decodeGetUserTierLimitsRequest(b)
		require.NoError(t, err)
		assert.Equal(t, "user-uuid-123", got)
	})

	t.Run("ignores unknown fields", func(t *testing.T) {
		var b []byte
		b = protowire.AppendTag(b, 2, protowire.VarintType) // unknown field
		b = protowire.AppendVarint(b, 42)
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, "u1")

		got, err := decodeGetUserTierLimitsRequest(b)
		require.NoError(t, err)
		assert.Equal(t, "u1", got)
	})

	t.Run("empty bytes returns empty string", func(t *testing.T) {
		got, err := decodeGetUserTierLimitsRequest(nil)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

// TestEncodeTierLimitsResponse memverifikasi setiap field number + wire type
// PERSIS match dengan decoder di api-gateway. Kalau test ini fail, contract
// dengan api-gateway berubah dan akan memutus integration.
func TestEncodeTierLimitsResponse(t *testing.T) {
	limits := &entity.PlanLimits{
		MaxTransactionsPerMonth:   500,
		MaxFinancialAccounts:      10,
		MaxInvestmentInstruments:  5,
		HistoryRetentionMonths:    12,
		EmailNotificationsEnabled: true,
		ExportCsvEnabled:          true,
	}
	b := encodeTierLimitsResponse(limits)

	// Decode lalu pastikan tiap field dimuat di nomor yang benar.
	got := parseResponseForTest(t, b)
	assert.Equal(t, int32(500), got.MaxTransactionsPerMonth)
	assert.Equal(t, int32(10), got.MaxFinancialAccounts)
	assert.Equal(t, int32(5), got.MaxInvestmentInstruments)
	assert.Equal(t, int32(12), got.HistoryRetentionMonths)
	assert.True(t, got.EmailNotificationsEnabled)
	assert.True(t, got.ExportCsvEnabled)
}

func TestEncodeTierLimitsResponse_FreeTier(t *testing.T) {
	// FREE tier: limits ada yang 0 dan false — pastikan tetap encoded (proto3 default still emitted di manual encoder ini).
	limits := &entity.PlanLimits{
		MaxTransactionsPerMonth:   50,
		MaxFinancialAccounts:      3,
		MaxInvestmentInstruments:  0,
		HistoryRetentionMonths:    3,
		EmailNotificationsEnabled: false,
		ExportCsvEnabled:          false,
	}
	b := encodeTierLimitsResponse(limits)
	got := parseResponseForTest(t, b)
	assert.Equal(t, int32(50), got.MaxTransactionsPerMonth)
	assert.Equal(t, int32(0), got.MaxInvestmentInstruments)
	assert.False(t, got.EmailNotificationsEnabled)
	assert.False(t, got.ExportCsvEnabled)
}

func TestEncodeTierLimitsResponse_PROUnlimited(t *testing.T) {
	// PRO tier pakai -1 → encoded sebagai unsigned int32 max (varint 2-complement).
	limits := &entity.PlanLimits{
		MaxTransactionsPerMonth:   -1,
		MaxFinancialAccounts:      -1,
		MaxInvestmentInstruments:  -1,
		HistoryRetentionMonths:    -1,
		EmailNotificationsEnabled: true,
		ExportCsvEnabled:          true,
	}
	b := encodeTierLimitsResponse(limits)
	got := parseResponseForTest(t, b)
	assert.Equal(t, int32(-1), got.MaxTransactionsPerMonth)
	assert.Equal(t, int32(-1), got.MaxFinancialAccounts)
}

type tierLimitsForTest struct {
	MaxTransactionsPerMonth   int32
	MaxFinancialAccounts      int32
	MaxInvestmentInstruments  int32
	HistoryRetentionMonths    int32
	EmailNotificationsEnabled bool
	ExportCsvEnabled          bool
}

// parseResponseForTest mereplikasi decoder client di api-gateway secara berdiri-sendiri.
// Tujuan: kalau encoder berubah field number/wire type, test ini akan fail.
func parseResponseForTest(t *testing.T, b []byte) tierLimitsForTest {
	t.Helper()
	var resp tierLimitsForTest
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		require.GreaterOrEqual(t, n, 0)
		b = b[n:]

		if typ == protowire.VarintType {
			v, n := protowire.ConsumeVarint(b)
			require.GreaterOrEqual(t, n, 0)
			b = b[n:]
			switch num {
			case 1:
				resp.MaxTransactionsPerMonth = int32(v)
			case 2:
				resp.MaxFinancialAccounts = int32(v)
			case 3:
				resp.MaxInvestmentInstruments = int32(v)
			case 4:
				resp.HistoryRetentionMonths = int32(v)
			case 5:
				resp.EmailNotificationsEnabled = v != 0
			case 6:
				resp.ExportCsvEnabled = v != 0
			}
		} else {
			n := protowire.ConsumeFieldValue(num, typ, b)
			require.GreaterOrEqual(t, n, 0)
			b = b[n:]
		}
	}
	return resp
}
