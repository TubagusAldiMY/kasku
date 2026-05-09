package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/encoding/protowire"
)

func init() {
	// Daftarkan rawServerCodec sebagai codec "proto" sebelum server apapun dibuat.
	// Ini menggantikan codec proto default sehingga server bisa menerima raw bytes
	// dari api-gateway yang menggunakan grpc.ForceCodec(rawBytesCodec{}) di sisi client.
	encoding.RegisterCodec(rawServerCodec{})
}

// rawServerCodec adalah server-side codec yang meneruskan raw bytes tanpa transformasi.
// Nama "proto" diperlukan agar codec ini menggantikan default proto codec pada server.
// Simetris dengan rawBytesCodec di api-gateway/proto/billing/v1/billing_grpc.go.
type rawServerCodec struct{}

func (rawServerCodec) Name() string { return "proto" }

func (rawServerCodec) Marshal(v interface{}) ([]byte, error) {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return nil, fmt.Errorf("rawServerCodec: tipe tidak didukung %T", v)
	}
	return m.data, nil
}

func (rawServerCodec) Unmarshal(data []byte, v interface{}) error {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return fmt.Errorf("rawServerCodec: tipe tidak didukung %T", v)
	}
	m.data = data
	return nil
}

// rawBytesMsg adalah container untuk raw protobuf wire-format bytes.
// Implementasi interface minimal agar bisa digunakan dengan grpc.ServiceDesc handler.
type rawBytesMsg struct {
	data []byte
}

// TierLimitsProvider mendefinisikan kontrak use case untuk mengambil tier limits.
// Digunakan sebagai abstraksi agar gRPC layer tidak bergantung pada implementasi konkrit.
type TierLimitsProvider interface {
	Execute(ctx context.Context, userID string) (*entity.PlanLimits, error)
}

// BillingGRPCServer mengimplementasikan billing.v1.BillingInternal gRPC service.
// Encoding/decoding dilakukan secara manual menggunakan protowire karena tidak ada file .proto
// yang di-generate — kompatibel penuh dengan api-gateway yang menggunakan rawBytesCodec.
type BillingGRPCServer struct {
	tierLimitsUC TierLimitsProvider
	log          zerolog.Logger
	server       *grpc.Server
}

// NewBillingGRPCServer membuat instance BillingGRPCServer baru.
func NewBillingGRPCServer(tierLimitsUC TierLimitsProvider, log zerolog.Logger) *BillingGRPCServer {
	return &BillingGRPCServer{
		tierLimitsUC: tierLimitsUC,
		log:          log,
	}
}

// Start memulai gRPC server pada port yang diberikan.
// Server berjalan di goroutine terpisah — panggil Stop() untuk graceful shutdown.
func (s *BillingGRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("gagal listen pada gRPC port %s: %w", port, err)
	}

	s.server = grpc.NewServer()
	s.server.RegisterService(&billingInternalServiceDesc, s)

	s.log.Info().Str("port", port).Msg("billing gRPC server listening")

	go func() {
		if err := s.server.Serve(lis); err != nil {
			s.log.Error().Err(err).Msg("billing gRPC server berhenti dengan error")
		}
	}()

	return nil
}

// Stop menghentikan gRPC server dengan graceful shutdown.
// Akan menunggu semua RPC in-flight selesai sebelum menutup listener.
func (s *BillingGRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

// billingInternalServiceDesc adalah deskriptor manual untuk service billing.v1.BillingInternal.
// Harus cocok dengan full method name yang digunakan di api-gateway:
// /billing.v1.BillingInternal/GetUserTierLimits
var billingInternalServiceDesc = grpc.ServiceDesc{
	ServiceName: "billing.v1.BillingInternal",
	HandlerType: (*BillingGRPCServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetUserTierLimits",
			Handler:    getUserTierLimitsHandler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

// getUserTierLimitsHandler adalah unary handler untuk RPC GetUserTierLimits.
// Signature harus sesuai persis dengan grpc.MethodDesc.Handler type.
func getUserTierLimitsHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	reqMsg := &rawBytesMsg{}
	if err := dec(reqMsg); err != nil {
		return nil, fmt.Errorf("gagal decode request: %w", err)
	}

	s, ok := srv.(*BillingGRPCServer)
	if !ok {
		return nil, fmt.Errorf("internal error: server type assertion gagal")
	}

	userID, err := decodeGetUserTierLimitsRequest(reqMsg.data)
	if err != nil {
		return nil, fmt.Errorf("gagal decode proto request: %w", err)
	}

	limits, err := s.tierLimitsUC.Execute(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Str("user_id", userID).Msg("gagal mengambil tier limits")
		return nil, err
	}

	respData := encodeTierLimitsResponse(limits)
	return &rawBytesMsg{data: respData}, nil
}

// decodeGetUserTierLimitsRequest mendecode request proto3 wire-format:
//
//	field 1 (BytesType): user_id (string)
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
			// Skip field yang tidak dikenal agar forward-compatible
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
// Urutan field dan nomor HARUS sinkron dengan decodeResponse di api-gateway/proto/billing/v1/billing_grpc.go.
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
