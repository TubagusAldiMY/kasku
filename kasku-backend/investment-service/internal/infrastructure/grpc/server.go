// Package grpc mengimplementasikan gRPC server internal untuk investment-service.
// Field numbers HARUS sinkron dengan proto structs di sync-service/src/proto/.
package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/TubagusAldiMY/kasku/investment-service/internal/infrastructure/persistence"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/encoding/protowire"
)

const grpcGracefulStopTimeout = 25 * time.Second

func init() {
	encoding.RegisterCodec(rawServerCodec{})
}

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
	ServerData []byte // reserved — tidak diisi oleh investment-service (field 3 di proto)
}

type entityChange struct {
	EntityID    string
	Operation   string
	Data        []byte
	UpdatedAtMs int64
}

func encodeSyncUpsertResult(r syncUpsertResult) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, r.SyncID)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, r.Status)
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

// InvestmentGRPCServer mengimplementasikan InvestmentInternal gRPC service.
type InvestmentGRPCServer struct {
	pool   *pgxpool.Pool
	log    zerolog.Logger
	server *grpc.Server
}

func NewInvestmentGRPCServer(pool *pgxpool.Pool, log zerolog.Logger) *InvestmentGRPCServer {
	return &InvestmentGRPCServer{pool: pool, log: log}
}

func (s *InvestmentGRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("gagal listen gRPC port %s: %w", port, err)
	}
	s.server = grpc.NewServer()
	s.server.RegisterService(&investmentInternalDesc, s)
	s.log.Info().Str("port", port).Msg("investment gRPC server listening")
	go func() {
		if err := s.server.Serve(lis); err != nil {
			s.log.Error().Err(err).Msg("investment gRPC server error")
		}
	}()
	return nil
}

func (s *InvestmentGRPCServer) Stop() {
	if s.server == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(grpcGracefulStopTimeout):
		s.log.Warn().Msg("investment gRPC GracefulStop timeout, memaksa Stop()")
		s.server.Stop()
	}
}

// investmentInternalServer is a marker interface satisfying grpc.ServiceDesc.HandlerType.
type investmentInternalServer any

var investmentInternalDesc = grpc.ServiceDesc{
	ServiceName: "investment.v1.InvestmentInternal",
	HandlerType: (*investmentInternalServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "UpsertInvestmentAssets", Handler: upsertInvestmentAssetsHandler},
		{MethodName: "ListInvestmentAssets", Handler: listInvestmentAssetsHandler},
	},
	Streams: []grpc.StreamDesc{},
}

func upsertInvestmentAssetsHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	msg := &rawBytesMsg{}
	if err := dec(msg); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	s := srv.(*InvestmentGRPCServer)
	req, err := decodeUpsertRequest(msg.data)
	if err != nil {
		return nil, fmt.Errorf("parse UpsertInvestmentAssets request: %w", err)
	}
	results, err := s.handleUpsert(ctx, req)
	if err != nil {
		return nil, err
	}
	return &rawBytesMsg{data: encodeUpsertResponse(results)}, nil
}

func listInvestmentAssetsHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	msg := &rawBytesMsg{}
	if err := dec(msg); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	s := srv.(*InvestmentGRPCServer)
	req, err := decodeListRequest(msg.data)
	if err != nil {
		return nil, fmt.Errorf("parse ListInvestmentAssets request: %w", err)
	}
	changes, err := s.handleList(ctx, req)
	if err != nil {
		return nil, err
	}
	return &rawBytesMsg{data: encodeListResponse(changes)}, nil
}

func (s *InvestmentGRPCServer) handleUpsert(ctx context.Context, req upsertReq) ([]syncUpsertResult, error) {
	if err := persistence.ValidateTenantSchema(req.TenantSchema); err != nil {
		return nil, fmt.Errorf("tenant schema tidak valid: %w", err)
	}
	results := make([]syncUpsertResult, 0, len(req.Items))
	for _, item := range req.Items {
		status := "applied"
		if err := s.applyOp(ctx, req.TenantSchema, item); err != nil {
			s.log.Error().Err(err).
				Str("sync_id", item.SyncID).
				Str("operation", item.Operation).
				Msg("gagal apply investment_asset sync")
			status = "error"
		}
		results = append(results, syncUpsertResult{SyncID: item.SyncID, Status: status})
	}
	return results, nil
}

func (s *InvestmentGRPCServer) applyOp(ctx context.Context, schema string, item syncUpsertItem) error {
	payload := string(item.Payload)
	switch item.Operation {
	case "create":
		q := fmt.Sprintf(`
			INSERT INTO %s.investment_assets
				(id, name, asset_type, symbol, quantity, avg_buy_price, currency, is_deleted, sort_order, created_at, updated_at)
			VALUES (
				$1::uuid,
				COALESCE(NULLIF($2::jsonb->>'name',''),'Untitled'),
				COALESCE(NULLIF($2::jsonb->>'asset_type',''),'STOCK'),
				COALESCE(NULLIF($2::jsonb->>'symbol',''),''),
				COALESCE(NULLIF($2::jsonb->>'quantity','')::numeric,0),
				COALESCE(NULLIF($2::jsonb->>'avg_buy_price','')::numeric,0),
				COALESCE(NULLIF($2::jsonb->>'currency',''),'IDR'),
				false,
				COALESCE(NULLIF($2::jsonb->>'sort_order','')::int,0),
				now(), now()
			)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name,
				asset_type=EXCLUDED.asset_type,
				symbol=EXCLUDED.symbol,
				quantity=EXCLUDED.quantity,
				avg_buy_price=EXCLUDED.avg_buy_price,
				currency=EXCLUDED.currency,
				sort_order=EXCLUDED.sort_order,
				is_deleted=false, deleted_at=NULL, updated_at=now()`, schema)
		_, err := s.pool.Exec(ctx, q, item.EntityID, payload)
		return err

	case "update":
		q := fmt.Sprintf(`
			UPDATE %s.investment_assets SET
				name=COALESCE(NULLIF($2::jsonb->>'name',''),name),
				asset_type=COALESCE(NULLIF($2::jsonb->>'asset_type',''),asset_type),
				symbol=COALESCE(NULLIF($2::jsonb->>'symbol',''),symbol),
				quantity=COALESCE(NULLIF($2::jsonb->>'quantity','')::numeric,quantity),
				avg_buy_price=COALESCE(NULLIF($2::jsonb->>'avg_buy_price','')::numeric,avg_buy_price),
				currency=COALESCE(NULLIF($2::jsonb->>'currency',''),currency),
				sort_order=COALESCE(NULLIF($2::jsonb->>'sort_order','')::int,sort_order),
				updated_at=now()
			WHERE id=$1::uuid AND is_deleted=false`, schema)
		_, err := s.pool.Exec(ctx, q, item.EntityID, payload)
		return err

	case "delete":
		q := fmt.Sprintf(`
			UPDATE %s.investment_assets
			SET is_deleted=true, deleted_at=now(), updated_at=now()
			WHERE id=$1::uuid AND is_deleted=false`, schema)
		_, err := s.pool.Exec(ctx, q, item.EntityID)
		return err

	default:
		return fmt.Errorf("operasi tidak dikenal: %s", item.Operation)
	}
}

func (s *InvestmentGRPCServer) handleList(ctx context.Context, req listReq) ([]entityChange, error) {
	if err := persistence.ValidateTenantSchema(req.TenantSchema); err != nil {
		return nil, fmt.Errorf("tenant schema tidak valid: %w", err)
	}
	q := fmt.Sprintf(`
		SELECT id::text,
		       to_jsonb(t.*)::text,
		       (EXTRACT(EPOCH FROM updated_at)*1000)::bigint,
		       CASE WHEN is_deleted THEN 'delete' ELSE 'upsert' END
		FROM %s.investment_assets t
		WHERE updated_at > to_timestamp($1::float8/1000)
		ORDER BY updated_at ASC`, req.TenantSchema)

	rows, err := s.pool.Query(ctx, q, req.SinceMs)
	if err != nil {
		return nil, fmt.Errorf("gagal query investment_assets: %w", err)
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
