// Package grpc mengimplementasikan gRPC server internal untuk transaction-service.
// Field numbers HARUS sinkron dengan proto structs di sync-service/src/proto/.
package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/TubagusAldiMY/kasku/transaction-service/internal/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	ServerData []byte // reserved — tidak diisi oleh transaction-service (field 3 di proto)
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

// TransactionGRPCServer mengimplementasikan TransactionInternal gRPC service.
type TransactionGRPCServer struct {
	pool   *pgxpool.Pool
	log    zerolog.Logger
	server *grpc.Server
}

func NewTransactionGRPCServer(pool *pgxpool.Pool, log zerolog.Logger) *TransactionGRPCServer {
	return &TransactionGRPCServer{pool: pool, log: log}
}

func (s *TransactionGRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("gagal listen gRPC port %s: %w", port, err)
	}
	s.server = grpc.NewServer()
	s.server.RegisterService(&transactionInternalDesc, s)
	s.log.Info().Str("port", port).Msg("transaction gRPC server listening")
	go func() {
		if err := s.server.Serve(lis); err != nil {
			s.log.Error().Err(err).Msg("transaction gRPC server error")
		}
	}()
	return nil
}

func (s *TransactionGRPCServer) Stop() {
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
		s.log.Warn().Msg("transaction gRPC GracefulStop timeout, memaksa Stop()")
		s.server.Stop()
	}
}

// transactionInternalServer is a marker interface satisfying grpc.ServiceDesc.HandlerType.
type transactionInternalServer any

var transactionInternalDesc = grpc.ServiceDesc{
	ServiceName: "transaction.v1.TransactionInternal",
	HandlerType: (*transactionInternalServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "UpsertTransactions", Handler: upsertTransactionsHandler},
		{MethodName: "ListTransactions", Handler: listTransactionsHandler},
	},
	Streams: []grpc.StreamDesc{},
}

func upsertTransactionsHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	msg := &rawBytesMsg{}
	if err := dec(msg); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	s := srv.(*TransactionGRPCServer)
	req, err := decodeUpsertRequest(msg.data)
	if err != nil {
		return nil, fmt.Errorf("parse UpsertTransactions request: %w", err)
	}
	results, err := s.handleUpsert(ctx, req)
	if err != nil {
		return nil, err
	}
	return &rawBytesMsg{data: encodeUpsertResponse(results)}, nil
}

func listTransactionsHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	msg := &rawBytesMsg{}
	if err := dec(msg); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	s := srv.(*TransactionGRPCServer)
	req, err := decodeListRequest(msg.data)
	if err != nil {
		return nil, fmt.Errorf("parse ListTransactions request: %w", err)
	}
	changes, err := s.handleList(ctx, req)
	if err != nil {
		return nil, err
	}
	return &rawBytesMsg{data: encodeListResponse(changes)}, nil
}

func (s *TransactionGRPCServer) handleUpsert(ctx context.Context, req upsertReq) ([]syncUpsertResult, error) {
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
				Msg("gagal apply transaction sync")
			status = "error"
		}
		results = append(results, syncUpsertResult{SyncID: item.SyncID, Status: status})
	}
	return results, nil
}

func (s *TransactionGRPCServer) applyOp(ctx context.Context, schema, userID string, item syncUpsertItem) error {
	payload := string(item.Payload)
	switch item.Operation {
	case "create":
		var p struct {
			AccountID       string `json:"account_id"`
			BudgetID        string `json:"budget_id"`
			ToAccountID     string `json:"to_account_id"`
			TransactionType string `json:"transaction_type"`
			AmountIDR       int64  `json:"amount_idr"`
		}
		if err := json.Unmarshal(item.Payload, &p); err != nil {
			return fmt.Errorf("gagal parse payload: %w", err)
		}
		accountID, err := uuid.Parse(p.AccountID)
		if err != nil {
			return fmt.Errorf("account_id tidak valid: %w", err)
		}
		var toAccountID *uuid.UUID
		if p.ToAccountID != "" {
			if parsed, parseErr := uuid.Parse(p.ToAccountID); parseErr == nil {
				toAccountID = &parsed
			}
		}
		var budgetID *uuid.UUID
		if p.TransactionType == "EXPENSE" && p.BudgetID != "" {
			parsed, parseErr := uuid.Parse(p.BudgetID)
			if parseErr != nil {
				return fmt.Errorf("budget_id tidak valid: %w", parseErr)
			}
			budgetID = &parsed
		}

		dbTx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("gagal mulai DB transaction: %w", err)
		}
		defer dbTx.Rollback(ctx) //nolint:errcheck

		if err := persistence.ValidateAccountForUser(ctx, dbTx, schema, userID, accountID); err != nil {
			return err
		}
		if p.TransactionType == "TRANSFER" {
			if toAccountID == nil || *toAccountID == accountID {
				return fmt.Errorf("rekening tujuan transfer tidak valid")
			}
			if err := persistence.ValidateAccountForUser(ctx, dbTx, schema, userID, *toAccountID); err != nil {
				return err
			}
		} else {
			toAccountID = nil
		}

		// Validasi saldo cukup untuk TRANSFER sebelum insert
		if p.TransactionType == "TRANSFER" && p.AmountIDR > 0 {
			var currentBalance int64
			balanceQ := fmt.Sprintf(
				"SELECT balance FROM %s.financial_accounts WHERE id=$1 AND user_id=$2::uuid AND is_deleted=false",
				schema,
			)
			if err := dbTx.QueryRow(ctx, balanceQ, accountID, userID).Scan(&currentBalance); err != nil {
				return fmt.Errorf("gagal cek saldo rekening asal: %w", err)
			}
			if p.AmountIDR > currentBalance {
				return fmt.Errorf("saldo rekening tidak mencukupi: tersedia %d, dibutuhkan %d", currentBalance, p.AmountIDR)
			}
		}
		if err := persistence.ValidateBudgetForUser(ctx, dbTx, schema, userID, budgetID); err != nil {
			return err
		}

		q := fmt.Sprintf(`
			INSERT INTO %s.transactions
				(id, sync_id, account_id, category_id, budget_id, transaction_type, amount_idr, transaction_date, notes, to_account_id, is_deleted, created_at, updated_at)
			SELECT
				$1::uuid, $2::text,
				NULLIF($4::jsonb->>'account_id','')::uuid,
				NULLIF($4::jsonb->>'category_id','')::uuid,
				$5::uuid,
				COALESCE(NULLIF($4::jsonb->>'transaction_type',''),'EXPENSE'),
				COALESCE(NULLIF($4::jsonb->>'amount_idr','')::bigint,0),
				COALESCE(NULLIF($4::jsonb->>'transaction_date','')::date,CURRENT_DATE),
				NULLIF($4::jsonb->>'notes',''),
				CASE
					WHEN COALESCE(NULLIF($4::jsonb->>'transaction_type',''),'EXPENSE') = 'TRANSFER'
					THEN NULLIF($4::jsonb->>'to_account_id','')::uuid
					ELSE NULL
				END,
				false, now(), now()
			WHERE EXISTS (
				SELECT 1 FROM %s.financial_accounts
				WHERE id=NULLIF($4::jsonb->>'account_id','')::uuid
				  AND user_id=$3::uuid AND is_deleted=false
			)
			ON CONFLICT (id) DO NOTHING`, schema, schema)

		tag, err := dbTx.Exec(ctx, q, item.EntityID, item.SyncID, userID, payload, budgetID)
		if err != nil {
			return fmt.Errorf("gagal insert transaction via sync: %w", err)
		}

		if tag.RowsAffected() == 0 {
			return dbTx.Commit(ctx)
		}

		if err := persistence.RecalculateAccountBalance(ctx, dbTx, schema, accountID); err != nil {
			return err
		}
		if toAccountID != nil {
			if err := persistence.RecalculateAccountBalance(ctx, dbTx, schema, *toAccountID); err != nil {
				return err
			}
		}
		return dbTx.Commit(ctx)

	case "update":
		var p struct {
			AccountID       string `json:"account_id"`
			BudgetID        string `json:"budget_id"`
			ToAccountID     string `json:"to_account_id"`
			TransactionType string `json:"transaction_type"`
		}
		if err := json.Unmarshal(item.Payload, &p); err != nil {
			return fmt.Errorf("gagal parse payload: %w", err)
		}
		var budgetID *uuid.UUID
		if p.TransactionType == "EXPENSE" && p.BudgetID != "" {
			parsed, parseErr := uuid.Parse(p.BudgetID)
			if parseErr != nil {
				return fmt.Errorf("budget_id tidak valid: %w", parseErr)
			}
			budgetID = &parsed
		}
		newAccountID, err := uuid.Parse(p.AccountID)
		if err != nil {
			return fmt.Errorf("account_id tidak valid: %w", err)
		}
		var newToAccountID *uuid.UUID
		if p.TransactionType == "TRANSFER" {
			if p.ToAccountID == "" {
				return fmt.Errorf("rekening tujuan transfer tidak valid")
			}
			parsed, parseErr := uuid.Parse(p.ToAccountID)
			if parseErr != nil {
				return fmt.Errorf("to_account_id tidak valid: %w", parseErr)
			}
			newToAccountID = &parsed
		}

		dbTx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("gagal mulai DB transaction: %w", err)
		}
		defer dbTx.Rollback(ctx) //nolint:errcheck

		selectQ := fmt.Sprintf(`
			SELECT account_id, to_account_id
			FROM %s.transactions
			WHERE id=$1::uuid AND is_deleted=false
			  AND account_id IN (SELECT id FROM %s.financial_accounts WHERE user_id=$2::uuid)`,
			schema, schema)
		var oldAccountID uuid.UUID
		var oldToAccountID *uuid.UUID
		if err := dbTx.QueryRow(ctx, selectQ, item.EntityID, userID).Scan(&oldAccountID, &oldToAccountID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("gagal ambil detail transaksi untuk update: %w", err)
		}
		if err := persistence.ValidateAccountForUser(ctx, dbTx, schema, userID, newAccountID); err != nil {
			return err
		}
		if newToAccountID != nil {
			if *newToAccountID == newAccountID {
				return fmt.Errorf("rekening tujuan transfer tidak valid")
			}
			if err := persistence.ValidateAccountForUser(ctx, dbTx, schema, userID, *newToAccountID); err != nil {
				return err
			}
		}
		if err := persistence.ValidateBudgetForUser(ctx, dbTx, schema, userID, budgetID); err != nil {
			return err
		}

		q := fmt.Sprintf(`
			UPDATE %s.transactions SET
				account_id=COALESCE(NULLIF($3::jsonb->>'account_id','')::uuid,account_id),
				category_id=CASE
					WHEN COALESCE(NULLIF($3::jsonb->>'transaction_type',''),transaction_type) = 'TRANSFER'
					THEN NULL
					ELSE NULLIF($3::jsonb->>'category_id','')::uuid
				END,
				budget_id=CASE
					WHEN COALESCE(NULLIF($3::jsonb->>'transaction_type',''),transaction_type) = 'EXPENSE'
					THEN NULLIF($3::jsonb->>'budget_id','')::uuid
					ELSE NULL
				END,
				transaction_type=COALESCE(NULLIF($3::jsonb->>'transaction_type',''),transaction_type),
				amount_idr=COALESCE(NULLIF($3::jsonb->>'amount_idr','')::bigint,amount_idr),
				transaction_date=COALESCE(NULLIF($3::jsonb->>'transaction_date','')::date,transaction_date),
				notes=COALESCE(NULLIF($3::jsonb->>'notes',''),notes),
				to_account_id=CASE
					WHEN COALESCE(NULLIF($3::jsonb->>'transaction_type',''),transaction_type) = 'TRANSFER'
					THEN NULLIF($3::jsonb->>'to_account_id','')::uuid
					ELSE NULL
				END,
				updated_at=now()
			WHERE id=$1::uuid AND is_deleted=false
			  AND account_id IN (
			    SELECT id FROM %s.financial_accounts WHERE user_id=$2::uuid
			  )`, schema, schema)
		if _, err := dbTx.Exec(ctx, q, item.EntityID, userID, payload); err != nil {
			return err
		}

		affected := []uuid.UUID{oldAccountID, newAccountID}
		if oldToAccountID != nil {
			affected = append(affected, *oldToAccountID)
		}
		if newToAccountID != nil {
			affected = append(affected, *newToAccountID)
		}
		seen := map[uuid.UUID]struct{}{}
		for _, accountID := range affected {
			if accountID == uuid.Nil {
				continue
			}
			if _, ok := seen[accountID]; ok {
				continue
			}
			seen[accountID] = struct{}{}
			if err := persistence.RecalculateAccountBalance(ctx, dbTx, schema, accountID); err != nil {
				return err
			}
		}
		return dbTx.Commit(ctx)

	case "delete":
		dbTx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("gagal mulai DB transaction: %w", err)
		}
		defer dbTx.Rollback(ctx) //nolint:errcheck

		selectQ := fmt.Sprintf(`
			SELECT account_id, to_account_id
			FROM %s.transactions
			WHERE id=$1::uuid AND is_deleted=false
			  AND account_id IN (SELECT id FROM %s.financial_accounts WHERE user_id=$2::uuid)`,
			schema, schema)

		var accountID uuid.UUID
		var toAccountID *uuid.UUID
		err = dbTx.QueryRow(ctx, selectQ, item.EntityID, userID).Scan(&accountID, &toAccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return dbTx.Commit(ctx)
		}
		if err != nil {
			return fmt.Errorf("gagal ambil detail transaksi untuk delete: %w", err)
		}

		deleteQ := fmt.Sprintf(`
			UPDATE %s.transactions
			SET is_deleted=true, deleted_at=now(), updated_at=now()
			WHERE id=$1::uuid AND is_deleted=false`, schema)
		if _, err := dbTx.Exec(ctx, deleteQ, item.EntityID); err != nil {
			return fmt.Errorf("gagal soft delete via sync: %w", err)
		}

		if err := persistence.RecalculateAccountBalance(ctx, dbTx, schema, accountID); err != nil {
			return err
		}
		if toAccountID != nil {
			if err := persistence.RecalculateAccountBalance(ctx, dbTx, schema, *toAccountID); err != nil {
				return err
			}
		}
		return dbTx.Commit(ctx)

	default:
		return fmt.Errorf("operasi tidak dikenal: %s", item.Operation)
	}
}

func (s *TransactionGRPCServer) handleList(ctx context.Context, req listReq) ([]entityChange, error) {
	if err := persistence.ValidateTenantSchema(req.TenantSchema); err != nil {
		return nil, fmt.Errorf("tenant schema tidak valid: %w", err)
	}
	q := fmt.Sprintf(`
		SELECT id::text,
		       to_jsonb(t.*)::text,
		       (EXTRACT(EPOCH FROM updated_at)*1000)::bigint,
		       CASE WHEN is_deleted THEN 'delete' ELSE 'upsert' END
		FROM %s.transactions t
		WHERE updated_at > to_timestamp($1::float8/1000)
		ORDER BY updated_at ASC`, req.TenantSchema)

	rows, err := s.pool.Query(ctx, q, req.SinceMs)
	if err != nil {
		return nil, fmt.Errorf("gagal query transactions: %w", err)
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
