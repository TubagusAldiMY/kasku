// Package grpc mengimplementasikan gRPC server internal untuk finance-service.
// Digunakan oleh sync-service untuk batch upsert dan list perubahan financial accounts.
//
// Wire encoding dilakukan manual via protowire (tidak ada protoc) agar simetris
// dengan tonic/prost client di sync-service.
// Field numbers HARUS sinkron dengan proto structs di sync-service/src/proto/.
package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/internal/infrastructure/persistence"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/encoding/protowire"
)

// grpcGracefulStopTimeout adalah batas waktu GracefulStop sebelum dipaksa Stop().
// Harus lebih kecil dari gracefulShutdownTimeout di main.go (30s) agar HTTP server
// masih punya waktu cukup untuk shutdown setelah gRPC selesai.
const grpcGracefulStopTimeout = 25 * time.Second

func init() {
	encoding.RegisterCodec(rawServerCodec{})
}

// ─── Raw bytes codec ─────────────────────────────────────────────────────────

type rawServerCodec struct{}

func (rawServerCodec) Name() string { return "proto" }

func (rawServerCodec) Marshal(v any) ([]byte, error) {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return nil, fmt.Errorf("rawServerCodec: unsupported type %T", v)
	}
	return m.data, nil
}

func (rawServerCodec) Unmarshal(data []byte, v any) error {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return fmt.Errorf("rawServerCodec: unsupported type %T", v)
	}
	m.data = data
	return nil
}

type rawBytesMsg struct{ data []byte }

// ─── Internal message types ───────────────────────────────────────────────────

type syncUpsertItem struct {
	SyncID     string
	EntityID   string
	Operation  string
	Payload    []byte
	ClientTsMs int64
}

type syncUpsertResult struct {
	SyncID     string
	Status     string // "applied" | "error"
	ServerData []byte
}

type entityChange struct {
	EntityID    string
	Operation   string // "upsert" | "delete"
	Data        []byte // JSON bytes
	UpdatedAtMs int64
}

// ─── Protowire encode helpers ─────────────────────────────────────────────────

func encodeSyncUpsertResult(r syncUpsertResult) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, r.SyncID)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, r.Status)
	if len(r.ServerData) > 0 {
		b = protowire.AppendTag(b, 3, protowire.BytesType)
		b = protowire.AppendBytes(b, r.ServerData)
	}
	return b
}

func encodeEntityChange(c entityChange) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, c.EntityID)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, c.Operation)
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendBytes(b, c.Data)
	b = protowire.AppendTag(b, 4, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(c.UpdatedAtMs))
	return b
}

// ─── Protowire decode helpers ─────────────────────────────────────────────────

func decodeSyncUpsertItem(data []byte) (syncUpsertItem, error) {
	var item syncUpsertItem
	b := data
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return item, protowire.ParseError(n)
		}
		b = b[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return item, protowire.ParseError(n)
			}
			item.SyncID = s
			b = b[n:]
		case num == 2 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return item, protowire.ParseError(n)
			}
			item.EntityID = s
			b = b[n:]
		case num == 3 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return item, protowire.ParseError(n)
			}
			item.Operation = s
			b = b[n:]
		case num == 4 && typ == protowire.BytesType:
			bs, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return item, protowire.ParseError(n)
			}
			item.Payload = append([]byte(nil), bs...)
			b = b[n:]
		case num == 5 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return item, protowire.ParseError(n)
			}
			item.ClientTsMs = int64(v)
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return item, protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return item, nil
}

// ─── UpsertFinancialAccounts request/response ─────────────────────────────────

type upsertReq struct {
	TenantSchema string
	UserID       string
	Items        []syncUpsertItem
}

func decodeUpsertRequest(data []byte) (upsertReq, error) {
	var req upsertReq
	b := data
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return req, protowire.ParseError(n)
		}
		b = b[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return req, protowire.ParseError(n)
			}
			req.TenantSchema = s
			b = b[n:]
		case num == 2 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return req, protowire.ParseError(n)
			}
			req.UserID = s
			b = b[n:]
		case num == 3 && typ == protowire.BytesType:
			bs, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return req, protowire.ParseError(n)
			}
			item, err := decodeSyncUpsertItem(bs)
			if err != nil {
				return req, err
			}
			req.Items = append(req.Items, item)
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return req, protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return req, nil
}

func encodeUpsertResponse(results []syncUpsertResult) []byte {
	var b []byte
	for _, r := range results {
		rb := encodeSyncUpsertResult(r)
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendBytes(b, rb)
	}
	return b
}

// ─── ListFinancialAccounts request/response ───────────────────────────────────

type listReq struct {
	TenantSchema string
	SinceMs      int64
}

func decodeListRequest(data []byte) (listReq, error) {
	var req listReq
	b := data
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return req, protowire.ParseError(n)
		}
		b = b[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return req, protowire.ParseError(n)
			}
			req.TenantSchema = s
			b = b[n:]
		case num == 2 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return req, protowire.ParseError(n)
			}
			req.SinceMs = int64(v)
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return req, protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return req, nil
}

func encodeListResponse(changes []entityChange) []byte {
	var b []byte
	for _, c := range changes {
		cb := encodeEntityChange(c)
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendBytes(b, cb)
	}
	return b
}

// ─── gRPC Server ──────────────────────────────────────────────────────────────

// FinanceGRPCServer implementasi gRPC internal untuk sync-service.
type FinanceGRPCServer struct {
	pool   *pgxpool.Pool
	log    zerolog.Logger
	server *grpc.Server
}

func NewFinanceGRPCServer(pool *pgxpool.Pool, log zerolog.Logger) *FinanceGRPCServer {
	return &FinanceGRPCServer{pool: pool, log: log}
}

func (s *FinanceGRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("gagal listen gRPC port %s: %w", port, err)
	}
	s.server = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	s.server.RegisterService(&financeInternalDesc, s)
	s.log.Info().Str("port", port).Msg("finance gRPC server listening")
	go func() {
		if err := s.server.Serve(lis); err != nil {
			s.log.Error().Err(err).Msg("finance gRPC server error")
		}
	}()
	return nil
}

func (s *FinanceGRPCServer) Stop() {
	if s.server == nil {
		return
	}
	// GracefulStop menunggu semua RPC aktif selesai, tapi bisa hang selamanya
	// jika ada long-running call. Berikan batas waktu eksplisit, lalu paksa Stop().
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(grpcGracefulStopTimeout):
		s.log.Warn().Msg("finance gRPC GracefulStop timeout, memaksa Stop()")
		s.server.Stop()
	}
}

// financeInternalServer is a marker interface satisfying grpc.ServiceDesc.HandlerType —
// must be an interface pointer, not a concrete type.
type financeInternalServer any

var financeInternalDesc = grpc.ServiceDesc{
	ServiceName: "finance.v1.FinanceInternal",
	HandlerType: (*financeInternalServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "UpsertFinancialAccounts", Handler: upsertFinancialAccountsHandler},
		{MethodName: "ListFinancialAccounts", Handler: listFinancialAccountsHandler},
	},
	Streams: []grpc.StreamDesc{},
}

func upsertFinancialAccountsHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	msg := &rawBytesMsg{}
	if err := dec(msg); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	s := srv.(*FinanceGRPCServer)
	req, err := decodeUpsertRequest(msg.data)
	if err != nil {
		return nil, fmt.Errorf("parse UpsertFinancialAccounts request: %w", err)
	}
	results, err := s.handleUpsert(ctx, req)
	if err != nil {
		return nil, err
	}
	return &rawBytesMsg{data: encodeUpsertResponse(results)}, nil
}

func listFinancialAccountsHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	msg := &rawBytesMsg{}
	if err := dec(msg); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	s := srv.(*FinanceGRPCServer)
	req, err := decodeListRequest(msg.data)
	if err != nil {
		return nil, fmt.Errorf("parse ListFinancialAccounts request: %w", err)
	}
	changes, err := s.handleList(ctx, req)
	if err != nil {
		return nil, err
	}
	return &rawBytesMsg{data: encodeListResponse(changes)}, nil
}

// ─── Business logic ───────────────────────────────────────────────────────────

func (s *FinanceGRPCServer) handleUpsert(ctx context.Context, req upsertReq) ([]syncUpsertResult, error) {
	if err := persistence.ValidateTenantSchema(req.TenantSchema); err != nil {
		return nil, fmt.Errorf("tenant schema tidak valid: %w", err)
	}
	results := make([]syncUpsertResult, 0, len(req.Items))
	for _, item := range req.Items {
		status := "applied"
		if err := s.applyOp(ctx, req.TenantSchema, req.UserID, item); err != nil {
			s.log.Error().Err(err).
				Str("sync_id", item.SyncID).
				Str("operation", item.Operation).
				Msg("gagal apply financial_account sync")
			status = "error"
		}
		results = append(results, syncUpsertResult{SyncID: item.SyncID, Status: status})
	}
	return results, nil
}

func (s *FinanceGRPCServer) applyOp(ctx context.Context, schema, userID string, item syncUpsertItem) error {
	payload := string(item.Payload)
	switch item.Operation {
	case "create":
		q := fmt.Sprintf(`
			INSERT INTO %s.financial_accounts
				(id, user_id, name, account_type, balance, currency, color, icon, is_deleted, created_at, updated_at)
			VALUES (
				$1::uuid, $2::uuid,
				COALESCE(NULLIF($3::jsonb->>'name',''),'Untitled'),
				COALESCE(NULLIF($3::jsonb->>'account_type',''),'BANK'),
				COALESCE(NULLIF($3::jsonb->>'balance','')::bigint,0),
				COALESCE(NULLIF($3::jsonb->>'currency',''),'IDR'),
				COALESCE(NULLIF($3::jsonb->>'color',''),'#6366f1'),
				COALESCE(NULLIF($3::jsonb->>'icon',''),'wallet'),
				false, now(), now()
			)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, account_type=EXCLUDED.account_type,
				balance=EXCLUDED.balance, currency=EXCLUDED.currency,
				color=EXCLUDED.color, icon=EXCLUDED.icon,
				is_deleted=false, deleted_at=NULL, updated_at=now()`, schema)
		_, err := s.pool.Exec(ctx, q, item.EntityID, userID, payload)
		return err

	case "update":
		q := fmt.Sprintf(`
			UPDATE %s.financial_accounts SET
				name=COALESCE(NULLIF($3::jsonb->>'name',''),name),
				account_type=COALESCE(NULLIF($3::jsonb->>'account_type',''),account_type),
				balance=COALESCE(NULLIF($3::jsonb->>'balance','')::bigint,balance),
				currency=COALESCE(NULLIF($3::jsonb->>'currency',''),currency),
				color=COALESCE(NULLIF($3::jsonb->>'color',''),color),
				icon=COALESCE(NULLIF($3::jsonb->>'icon',''),icon),
				updated_at=now()
			WHERE id=$1::uuid AND user_id=$2::uuid AND is_deleted=false`, schema)
		_, err := s.pool.Exec(ctx, q, item.EntityID, userID, payload)
		return err

	case "delete":
		q := fmt.Sprintf(`
			UPDATE %s.financial_accounts
			SET is_deleted=true, deleted_at=now(), updated_at=now()
			WHERE id=$1::uuid AND user_id=$2::uuid AND is_deleted=false`, schema)
		_, err := s.pool.Exec(ctx, q, item.EntityID, userID)
		return err

	default:
		return fmt.Errorf("operasi tidak dikenal: %s", item.Operation)
	}
}

func (s *FinanceGRPCServer) handleList(ctx context.Context, req listReq) ([]entityChange, error) {
	if err := persistence.ValidateTenantSchema(req.TenantSchema); err != nil {
		return nil, fmt.Errorf("tenant schema tidak valid: %w", err)
	}
	q := fmt.Sprintf(`
		SELECT id::text,
		       to_jsonb(t.*)::text,
		       (EXTRACT(EPOCH FROM updated_at)*1000)::bigint,
		       CASE WHEN is_deleted THEN 'delete' ELSE 'upsert' END
		FROM %s.financial_accounts t
		WHERE updated_at > to_timestamp($1::float8/1000)
		ORDER BY updated_at ASC`, req.TenantSchema)

	rows, err := s.pool.Query(ctx, q, req.SinceMs)
	if err != nil {
		return nil, fmt.Errorf("gagal query financial_accounts: %w", err)
	}
	defer rows.Close()

	var changes []entityChange
	for rows.Next() {
		var (
			entityID    string
			dataJSON    string
			updatedAtMs int64
			operation   string
		)
		if err := rows.Scan(&entityID, &dataJSON, &updatedAtMs, &operation); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		changes = append(changes, entityChange{
			EntityID:    entityID,
			Operation:   operation,
			Data:        []byte(dataJSON),
			UpdatedAtMs: updatedAtMs,
		})
	}
	return changes, rows.Err()
}
