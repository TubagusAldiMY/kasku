// Package billingv1 menyediakan gRPC client untuk billing-service.
// File ini ditulis manual karena protoc tidak tersedia di build environment.
// Wire encoding menggunakan google.golang.org/protobuf/encoding/protowire
// sehingga kompatibel penuh dengan server yang dibuild dari billing.proto.
package billingv1

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protowire"
)

// ─── Messages ────────────────────────────────────────────────────────────────

// GetUserTierLimitsRequest adalah request message untuk RPC GetUserTierLimits.
type GetUserTierLimitsRequest struct {
	UserID string // proto field 1: string user_id
}

// TierLimitsResponse adalah response message dari RPC GetUserTierLimits.
type TierLimitsResponse struct {
	MaxTransactionsPerMonth   int32 // field 1
	MaxFinancialAccounts      int32 // field 2
	MaxInvestmentInstruments  int32 // field 3
	HistoryRetentionMonths    int32 // field 4
	EmailNotificationsEnabled bool  // field 5
	ExportCsvEnabled          bool  // field 6
}

// ─── Wire encoding ────────────────────────────────────────────────────────────

func encodeRequest(req *GetUserTierLimitsRequest) []byte {
	if req.UserID == "" {
		return nil
	}
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, req.UserID)
	return b
}

func decodeResponse(b []byte) (*TierLimitsResponse, error) {
	resp := &TierLimitsResponse{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		b = b[n:]

		switch typ {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
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
		default:
			// skip unknown / future fields
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return resp, nil
}

// ─── Raw bytes codec ──────────────────────────────────────────────────────────

// rawBytesMsg adalah container untuk raw protobuf bytes yang dipakai bersama rawBytesCodec.
type rawBytesMsg struct{ data []byte }

// rawBytesCodec adalah custom gRPC codec yang hanya meneruskan raw bytes.
// Name() = "proto" sehingga server (yang menggunakan codec proto standar) bisa
// menginterpretasi bytes yang dikirim/diterima.
type rawBytesCodec struct{}

func (rawBytesCodec) Name() string { return "proto" }

func (rawBytesCodec) Marshal(v interface{}) ([]byte, error) {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return nil, fmt.Errorf("rawBytesCodec: expected *rawBytesMsg, got %T", v)
	}
	return m.data, nil
}

func (rawBytesCodec) Unmarshal(data []byte, v interface{}) error {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return fmt.Errorf("rawBytesCodec: expected *rawBytesMsg, got %T", v)
	}
	m.data = data
	return nil
}

// ─── Client interface ─────────────────────────────────────────────────────────

// BillingInternalClient adalah interface gRPC client untuk BillingInternal service.
type BillingInternalClient interface {
	GetUserTierLimits(ctx context.Context, req *GetUserTierLimitsRequest, opts ...grpc.CallOption) (*TierLimitsResponse, error)
}

// ─── Client implementation ────────────────────────────────────────────────────

type billingInternalClient struct {
	cc grpc.ClientConnInterface
}

// NewBillingInternalClient membuat instance billing gRPC client baru.
func NewBillingInternalClient(cc grpc.ClientConnInterface) BillingInternalClient {
	return &billingInternalClient{cc: cc}
}

func (c *billingInternalClient) GetUserTierLimits(
	ctx context.Context,
	req *GetUserTierLimitsRequest,
	opts ...grpc.CallOption,
) (*TierLimitsResponse, error) {
	reqMsg := &rawBytesMsg{data: encodeRequest(req)}
	respMsg := &rawBytesMsg{}

	callOpts := append([]grpc.CallOption{grpc.ForceCodec(rawBytesCodec{})}, opts...)

	if err := c.cc.Invoke(
		ctx,
		"/billing.v1.BillingInternal/GetUserTierLimits",
		reqMsg,
		respMsg,
		callOpts...,
	); err != nil {
		return nil, err
	}

	return decodeResponse(respMsg.data)
}
