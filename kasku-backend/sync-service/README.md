# sync-service

Offline sync engine untuk KasKu. Mengeksekusi batch upload (push) operasi offline dari client PWA
dan pull delta untuk meng-hidrasi cache client setelah reconnect. Konflik diselesaikan dengan
strategi **Server Wins** (ADR-012).

- Tech: Rust + Axum + sqlx + tonic (Prost)
- Port: HTTP **8088** (tidak menjalankan gRPC server)
- Database: `kasku_finance` (akses ke schema per-tenant)
- DB user: `kasku_sync_svc`

---

## Ownership & Boundary

sync-service **memiliki** dua endpoint berikut:

| Method | Path | Tujuan |
|--------|------|--------|
| POST | `/v1/sync/push` | Apply batch operasi offline (Server Wins) |
| GET | `/v1/sync/pull?since=ISO8601` | Delta perubahan sejak timestamp |
| GET | `/health` | Liveness + Postgres ping |
| GET | `/metrics` | Prometheus counters |

sync-service **tidak memiliki** schema database. `sync_log` table di-create di setiap tenant
schema oleh `ensure_tenant_runtime_objects()` yang di-deklarasi di
`finance-service/migrations/000004_create_sync_log_and_tenant_runtime_objects.up.sql`.
Karena itulah service ini tidak punya folder `migrations/`. Menempatkan migration di sini akan
men-duplikasi ownership.

---

## Arsitektur

```
[client offline]
      │ POST /v1/sync/push  ─►  api-gateway ──► sync-service
                                                    │
                                  (1) idempotency check (sync_id_exists)
                                  (2) conflict detection (server updated_at > client_ts ?)
                                  (3) gRPC fan-out ke owning service:
                                        - finance.v1.FinanceInternal/UpsertFinancialAccounts
                                        - transaction.v1.TransactionInternal/UpsertTransactions
                                        - investment.v1.InvestmentInternal/UpsertInvestmentAssets
                                  (4) log ke {tenant}.sync_log
```

Untuk **pull**, sync-service melakukan fan-out paralel via `tokio::join!` ke
`List*` RPC di tiga service yang sama, lalu menggabungkan hasilnya dan
mengurutkan berdasarkan `updated_at`.

### Trade-off Idempotency

Urutan saat ini adalah **apply-then-log**:

1. INSERT ke sync_log dilakukan setelah gRPC sukses
2. Jika gRPC sukses tapi INSERT log gagal, retry akan kembali memanggil gRPC

Trade-off ini disengaja:

- Owning service (finance/transaction/investment) **wajib idempotent** terhadap `item.sync_id` yang
  dikirim di payload `SyncUpsertItem`. Mereka adalah authority untuk dataset masing-masing.
- Pendekatan "claim-then-apply" akan butuh skema enum tambahan di `sync_log.operation`
  (misal `PENDING`/`ERROR`) yang saat ini tidak diizinkan oleh CHECK constraint di
  `finance-service/migrations/000004_*`.
- Untuk berpindah ke pre-call logging, butuh migration baru di finance-service + update
  CHECK constraint. Saat ini kontrak antar service sudah aman: duplicate sync_id di
  owning service akan ditolak/diterima sebagai no-op.

---

## gRPC Contract

sync-service adalah **client gRPC saja**. Server berada di:

| Service | gRPC addr (docker) | Methods |
|---------|--------------------|---------|
| finance-service | `kasku-finance-service:9084` | `UpsertFinancialAccounts`, `ListFinancialAccounts` |
| transaction-service | `kasku-transaction-service:9085` | `UpsertTransactions`, `ListTransactions` |
| investment-service | `kasku-investment-service:9086` | `UpsertInvestmentAssets`, `ListInvestmentAssets` |

Encoding pakai `tonic::codec::ProstCodec`. Go server pakai `rawServerCodec` hand-written via
`protowire` — kedua sisi tetap berbicara proto3 wire format, jadi inter-op aman.

Setiap RPC dibungkus `tokio::time::timeout(GRPC_REQUEST_TIMEOUT_MS, …)`. Timeout default 5000ms.
Tonic `Request::set_timeout()` juga di-set agar deadline di-propagasi ke server (`grpc-timeout` header).

---

## Dev Commands

```bash
make run            # cargo run
make build          # cargo build --release
make test           # cargo test
make lint           # cargo clippy -- -D warnings
make docker-build   # docker build -t kasku-sync-service:latest .
make clean          # cargo clean
```

Local dev butuh `kasku_finance` database hidup (jalankan via `docker compose up -d postgres`).
Jangan lupa `kasku_sync_svc` user sudah di-grant di `infra/postgres/00-init-databases.sh`.

---

## Environment Variables

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `HTTP_PORT` | `8088` | Port HTTP server |
| `DATABASE_URL` | _wajib_ | DSN ke `kasku_finance` (user `kasku_sync_svc`) |
| `APP_ENV` | `development` | Label environment (untuk log) |
| `LOG_LEVEL` | `info` | tracing-subscriber env filter |
| `FINANCE_SERVICE_GRPC_ADDR` | `http://localhost:9084` | gRPC endpoint finance-service |
| `TRANSACTION_SERVICE_GRPC_ADDR` | `http://localhost:9085` | gRPC endpoint transaction-service |
| `INVESTMENT_SERVICE_GRPC_ADDR` | `http://localhost:9086` | gRPC endpoint investment-service |
| `GRPC_REQUEST_TIMEOUT_MS` | `5000` | Hard timeout per RPC call (client-side) |

Lihat `.env.example` untuk template.

---

## Observability

- Structured JSON log via `tracing-subscriber` (lihat `main.rs`)
- `/metrics` mengekspos counter:
  - `kasku_sync_push_total`
  - `kasku_sync_push_conflicts_total`
  - `kasku_sync_push_skipped_total`
  - `kasku_sync_pull_total`
  - `kasku_sync_errors_total`
- Error response ke client di-sanitasi — internal address atau library trace tidak pernah
  dikembalikan ke caller. Detail tetap ada di log server dengan field terstruktur
  (`service`, `code`, `message`).

---

## Security Notes

- Tenant schema divalidasi dengan regex `^tenant_[0-9a-f_]{32,36}$` sebelum interpolasi ke query.
  Lihat `src/infrastructure/tenant.rs`.
- Ownership check: `tenant_schema` di header diverifikasi cocok dengan `user_id` (deterministic
  derivation, sama dengan auth-service JWT claim).
- Tidak ada PII di-log. `sync_id`, `entity_id`, `entity_type` di-log untuk audit; payload body tidak
  pernah di-log.
