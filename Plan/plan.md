# Rencana Implementasi — KasKu SaaS Backend (Phase 1 & 2)

## Context

auth-service dan api-gateway sudah selesai. Sembilan service berikutnya belum ada. Bottleneck utama:

- **user-service**: harus consume `user.registered` → panggil `provision_tenant()` di kasku_finance + INSERT subscription FREE ke kasku_billing. Tanpa ini, semua service lain tidak bisa beroperasi karena tenant schema belum ada.
- **billing-service**: api-gateway sudah memiliki gRPC client (`api-gateway/internal/infrastructure/grpc/billing_client.go`) yang memanggil `/billing.v1.BillingInternal/GetUserTierLimits`. Tanpa billing-service, api-gateway fallback ke FREE tier selamanya.
- **finance-service & transaction-service**: butuh tenant schema (dari user-service) + tier limit headers (dari api-gateway middleware yang sudah inject dari billing-service).

Urutan implementasi didiktekan oleh dependency tersebut.

---

## Fakta Kritis yang Diverifikasi dari Codebase

| Aspek | Nilai |
|-------|-------|
| Module path | `github.com/TubagusAldiMY/kasku/<service>` |
| HTTP framework | Gin + zerolog |
| DB driver | pgx/v5 + pgxpool, golang-migrate |
| Message broker | amqp091-go, exchange `kasku.events` (topic, durable) |
| JWT | RS256 (RSA keys — shared public key) |
| tenant_schema derivation | `"tenant_" + strings.ReplaceAll(user.ID.String(), "-", "_")` — already in auth-service JWT claims |
| Tier headers injected by api-gateway | `X-User-ID`, `X-Tenant-Schema`, `X-Subscription-Tier`, `X-Tier-Max-Transactions`, `X-Tier-Max-Accounts`, `X-Tier-Max-Investments`, `X-Tier-History-Months` |
| Billing gRPC proto | hand-written (no protoc), server path `/billing.v1.BillingInternal/GetUserTierLimits` |
| RabbitMQ queues | **TIDAK ada pre-declared service queue** (hanya DLQ). Setiap consumer WAJIB declare queue + bind saat startup |
| Port layout | gateway:8080, auth:8081/gRPC:9081, user:8082/gRPC:9082, billing:8083/gRPC:9083, finance:8084/gRPC:9084, tx:8085/gRPC:9085 |

---

## Dependency Graph

```
auth-service (done)
  │ publish user.registered
  ▼
user-service                    ← Phase 1A: Task 1
  │ SELECT provision_tenant()   (kasku_finance)
  │ INSERT subscription FREE    (kasku_billing)
  │
  ├─► billing-service           ← Phase 1A: Task 2
  │     │ gRPC GetUserTierLimits (port 9083)
  │     └─ consumed by api-gateway (already wired, fallback FREE today)
  │
  ├─► finance-service           ← Phase 1B: Task 3
  │     (kasku_finance, tenant schema, tier limit headers from gateway)
  │
  ├─► transaction-service       ← Phase 1B: Task 4
  │     (kasku_finance, tenant schema, tier limit headers from gateway)
  │
  └─► notification-service      ← Phase 1C: Task 5 (stateless, parallel)

Phase 2 (after Phase 1B):
  investment-service (Go)  ← Task 6
  price-service (Rust)     ← Task 7 (independent)
  sync-service (Rust)      ← Task 8 (needs finance/tx/investment gRPC)

Phase 4:
  admin-service            ← Task 9
```

---

## PHASE 1A — user-service + billing-service

### Task 1: user-service

**Direktori**: `kasku-backend/user-service/`  
**Pola**: Ikuti persis `auth-service/` (struktur direktori, config loading, graceful shutdown)  
**Database**: kasku_finance (DDL via `provision_tenant`) + kasku_billing (INSERT subscription)  
**DB user**: `kasku_user_svc` — sudah dibuat di `infra/postgres/00-init-databases.sh`, granted ke kasku_billing dan kasku_finance

#### Struktur file yang perlu dibuat
```
user-service/
  cmd/server/main.go
  configs/config.go
  internal/
    domain/entity/user_profile.go
    domain/errors/domain_errors.go
    delivery/http/handler/user_handler.go
    delivery/http/router.go
    infrastructure/
      messaging/rabbitmq_consumer.go   ← NEW pattern (bukan publisher)
      persistence/db.go
      persistence/postgres_user_repository.go
  go.mod
  Dockerfile
```

#### RabbitMQ Consumer Pattern
Tidak ada contoh consumer di codebase — ini perlu dibuat baru. Pola yang benar:

```go
// Saat startup: declare queue + bind ke exchange kasku.events
ch.QueueDeclare("kasku.user-service", true, false, false, false, amqp.Table{
    "x-dead-letter-exchange": "kasku.events.dlx",
})
ch.QueueBind("kasku.user-service", "user.registered", "kasku.events", false, nil)
msgs, _ := ch.Consume("kasku.user-service", "", false, false, false, false, nil)
// goroutine: for msg := range msgs { ... msg.Ack(false) }
```

#### Use Cases
1. **ProvisionTenantUseCase** — dipicu `user.registered` event:
    - Panggil `SELECT provision_tenant($1::uuid)` di kasku_finance (idempotent by design)
    - INSERT INTO kasku_billing.public.subscriptions ON CONFLICT DO NOTHING
    - tenant_schema sudah deterministik: `tenant_` + userID.Replace("-","_") — **tidak perlu dihitung ulang**, cukup validasi

#### HTTP Endpoints
- `GET /health` — ping kasku_finance + kasku_billing
- `GET /v1/users/profile` — baca dari JWT headers (`X-User-ID`, `X-Tenant-Schema`) — **tidak query DB**
- `PUT /v1/users/profile` — update username di kasku_auth.public.users via DSN terpisah (user-service perlu READ/WRITE ke kasku_auth untuk update username, atau hanya kasku_auth table via cross-DB query)

> **Keputusan desain**: user-service akses kasku_auth langsung dengan user `kasku_user_svc` yang perlu di-grant oleh infra. Alternatif: `PUT /v1/users/profile` hanya update display name di kasku_billing (bukan email/username di kasku_auth) untuk menghindari cross-DB coupling. **Rekomendasikan** opsi ini untuk Phase 1 — ubah `username` di kasku_billing.subscriptions atau buat tabel `user_profiles` di kasku_billing.

#### gRPC Server
Port 9082 — minimal health check via `grpc.NewServer()` + `health.RegisterHealthServer()`

#### docker-compose.yml — tambahkan:
```yaml
user-service:
  build: ./user-service
  container_name: kasku-user-service
  networks: [kasku-internal, kasku-data]
  environment:
    SERVER_PORT: "8082"
    GRPC_PORT: "9082"
    POSTGRES_FINANCE_DSN: ...
    POSTGRES_BILLING_DSN: ...
    RABBITMQ_URL: ...
```

---

### Task 2: billing-service

**Direktori**: `kasku-backend/billing-service/`  
**Database**: `kasku_billing` — tabel di public schema  
**DB user**: `kasku_billing_svc`

#### Migrations (di `billing-service/migrations/`)
```
000001_create_updated_at_function.up.sql  ← copy from auth-service
000002_create_subscription_plans.up.sql   ← + seed FREE/BASIC/PRO
000003_create_subscriptions.up.sql
000004_create_payments.up.sql
000005_create_indexes.up.sql
```

#### Tabel Kunci
```sql
-- subscription_plans
CREATE TABLE subscription_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(20) NOT NULL,          -- FREE, BASIC, PRO
  price_idr INTEGER NOT NULL DEFAULT 0,
  limits JSONB NOT NULL,              -- {"max_transactions":50, "max_accounts":3, ...}
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO subscription_plans (name, price_idr, limits) VALUES
  ('FREE', 0, '{"max_transactions_per_month":50,"max_financial_accounts":3,...}'),
  ('BASIC', 49000, '{"max_transactions_per_month":500,"max_financial_accounts":10,...}'),
  ('PRO', 99000, '{"max_transactions_per_month":-1,"max_financial_accounts":-1,...}');

-- subscriptions
CREATE TABLE subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL UNIQUE,
  plan_id UUID NOT NULL REFERENCES subscription_plans(id),
  status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  current_period_start TIMESTAMPTZ NOT NULL DEFAULT now(),
  current_period_end TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

#### gRPC Server (KRITIS — sudah ada client di api-gateway)

Client di api-gateway menggunakan `rawBytesCodec` dengan `grpc.ForceCodec` — ini hanya override sisi client. Server menerima dan mengirim standard proto3 binary encoding.

Karena protoc tidak tersedia, buat server handler secara manual menggunakan `google.golang.org/grpc` dengan `grpc.ServiceDesc`:

```go
// billing-service/internal/infrastructure/grpc/server.go
var billingServiceDesc = grpc.ServiceDesc{
    ServiceName: "billing.v1.BillingInternal",
    HandlerType: (*BillingInternalServer)(nil),
    Methods: []grpc.MethodDesc{
        {
            MethodName: "GetUserTierLimits",
            Handler:    _BillingInternal_GetUserTierLimits_Handler,
        },
    },
}
```

Decode request (field 1 = user_id string) dan encode response menggunakan `protowire` — persis seperti cara client di `billing_grpc.go` tapi terbalik.

Port gRPC: **9083**

#### HTTP Endpoints
- `GET /health`
- `GET /v1/billing/plans`
- `GET /v1/billing/subscription` (butuh `X-User-ID` header dari gateway)
- `POST /v1/billing/subscribe` → return HTTP 503 "coming soon" untuk Phase 1
- `POST /v1/billing/webhook/midtrans` → return HTTP 200 untuk Phase 1

#### Cron (in-process)
`time.NewTicker(1 * time.Hour)` — cek subscriptions expired → update status → publish `subscription.expired`

#### RabbitMQ Publisher (untuk cron)
Tambahkan event type `subscription.expired`, `subscription.expiring` ke publisher.

---

## PHASE 1B — finance-service + transaction-service

### Task 3: finance-service

**Direktori**: `kasku-backend/finance-service/`  
**Database**: `kasku_finance` — tenant schema per request  
**DB user**: `kasku_finance_svc`  
**Port**: HTTP 8084, gRPC 9084

#### Pola Tenant Schema
Semua repository method menerima `tenantSchema string`. Query format:
```go
fmt.Sprintf("SELECT * FROM %s.financial_accounts WHERE ...", tenantSchema)
```
> **Keamanan**: validasi tenantSchema dengan regex `^tenant_[0-9a-f_]{32,36}$` sebelum interpolasi ke query untuk cegah SQL injection.

#### Use Cases
1. `CreateAccountUseCase` — baca `X-Tier-Max-Accounts` header; jika `-1` = unlimited; jika limit tercapai → HTTP 402
2. `ListAccountsUseCase`
3. `GetAccountUseCase`
4. `UpdateAccountUseCase`
5. `UpdateBalanceUseCase` — UPDATE balance + INSERT balance_history (atomic transaction)
6. `DeleteAccountUseCase` — soft delete (`is_deleted=true`, `deleted_at=now()`)
7. `GetBalanceHistoryUseCase` — filter by `X-Tier-History-Months`

#### HTTP Endpoints
- `GET /health`
- `GET /v1/accounts`
- `POST /v1/accounts`
- `GET /v1/accounts/:id`
- `PUT /v1/accounts/:id`
- `PATCH /v1/accounts/:id/balance`
- `DELETE /v1/accounts/:id`
- `GET /v1/accounts/:id/history`

#### gRPC Server — Port 9084
Untuk sync-service (Phase 2): methods `UpsertFinancialAccounts`, `ListFinancialAccounts`

---

### Task 4: transaction-service

**Direktori**: `kasku-backend/transaction-service/`  
**Database**: `kasku_finance` — tenant schema  
**DB user**: `kasku_transaction_svc`  
**Port**: HTTP 8085, gRPC 9085

#### Domain Entities
- `Transaction` — id, sync_id (VARCHAR unique, untuk offline sync idempotency), account_id, category_id, transaction_type (INCOME/EXPENSE/TRANSFER), amount_idr, transaction_date, notes, to_account_id, is_deleted
- `Category` — id, name, icon, color, category_type (INCOME/EXPENSE/BOTH), is_default, is_deleted

#### Use Cases
1. `ListTransactionsUseCase` — filter date range + pagination + summary (total_income, total_expense, net); filter retention months dari `X-Tier-History-Months`
2. `CreateTransactionUseCase` — cek monthly quota via `X-Tier-Max-Transactions`; idempotency via `sync_id`; INSERT balance update di financial_accounts (via finance-service gRPC atau langsung ke tenant schema — **rekomendasikan** direct ke tenant schema untuk simplisitas Phase 1)
3. `UpdateTransactionUseCase`
4. `DeleteTransactionUseCase` — soft delete
5. `ListCategoriesUseCase` — include default categories (seed data)
6. `CreateCategoryUseCase`, `UpdateCategoryUseCase`, `DeleteCategoryUseCase` (cek ada transaksi aktif → 409)
7. `ExportCSVUseCase` — cek `X-Tier-Export-CSV` header (`export_csv_enabled` dari `X-Subscription-Tier` atau tambah header baru); baca dari `TierLimitsResponse.ExportCsvEnabled`

> **Catatan**: `ExportCsvEnabled` sudah ada di `TierLimitsResponse` (field 6) dan dikirim via api-gateway. Perlu tambah `X-Tier-Export-CSV` ke header injection di auth middleware api-gateway — ini **modifikasi kecil** di `api-gateway/internal/delivery/http/middleware/auth.go`.

#### HTTP Endpoints
- `GET /health`
- `GET|POST /v1/transactions`
- `GET|PUT|DELETE /v1/transactions/:id`
- `GET|POST /v1/categories`
- `PUT|DELETE /v1/categories/:id`
- `GET /v1/transactions/export`

---

## PHASE 1C — notification-service

### Task 5: notification-service

**Direktori**: `kasku-backend/notification-service/`  
**Database**: Tidak ada  
**Port**: HTTP 8086 (hanya health)

#### RabbitMQ Consumer
Queue `kasku.notification-service`, bind ke routing keys:
- `user.registered` → welcome email + verification link
- `user.email_verification_resent` → resend verification email
- `user.password_reset_requested` → reset password email
- `payment.succeeded` → receipt
- `payment.failed` → alert
- `subscription.expiring` → warning
- `subscription.expired` → downgrade notice
- `subscription.cancelled` → confirmation

#### Email Provider
SMTP via `net/smtp` standard library. Config: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`.

HTML templates di-embed via `//go:embed templates/*.html`.

#### Event Payload dari auth-service (sudah terdefinisi di `rabbitmq_publisher.go`)
- `UserRegisteredEvent{UserID, Email, Username, VerificationToken}`
- `EmailVerificationResentEvent{UserID, Email, VerificationToken}`
- `PasswordResetRequestedEvent{UserID, Email, ResetToken}`

---

## PHASE 2 — investment-service + price-service (Rust) + sync-service (Rust)

### Task 6: investment-service (Go)

**Port**: HTTP 8087, gRPC 9086  
**Database**: `kasku_finance` — tenant schema  
**DB user**: `kasku_investment_svc`

Use cases: CRUD InvestmentAsset + UnitHistory + fetch harga dari price-service via gRPC.  
Tier check: `X-Tier-Max-Investments` header.

---

### Task 7: price-service (Rust)

**Tech**: Axum + SQLx + Tonic + reqwest  
**Port**: HTTP 8088, gRPC 9087  
**Database**: `kasku_price`  
**DB user**: `kasku_price_svc`

External APIs: CoinGecko + metals.live  
**SSRF protection**: whitelist domain sebelum HTTP request.  
Cache TTL: 15 menit di DB.

---

### Task 8: sync-service (Rust)

**Tech**: Axum + SQLx + Tonic  
**Port**: HTTP 8089  
**DB user**: `kasku_sync_svc`

Endpoints: `POST /v1/sync/batch`, `GET /v1/sync/pull`  
Conflict resolution: Server Wins (jika sync_id sudah ada, kembalikan server state).

---

## PHASE 4 — admin-service

### Task 9: admin-service

**Port**: HTTP 8090 (network: kasku-admin only)  
**Database**: kasku_admin (R/W) + kasku_auth (R) + kasku_billing (R)  
**Auth**: HS256 dengan `ADMIN_JWT_SECRET` (terpisah dari user JWT)

---

## Modifikasi File yang Sudah Ada

### `api-gateway/internal/delivery/http/middleware/auth.go`
Tambah injection `X-Tier-Export-CSV` header:
```go
// Setelah baris HeaderTierHistoryMonths:
if limits.ExportCsvEnabled {
    c.Request.Header.Set("X-Tier-Export-CSV", "true")
} else {
    c.Request.Header.Set("X-Tier-Export-CSV", "false")
}
```

### `kasku-backend/docker-compose.yml`
Tambahkan service block untuk setiap service baru (user-service, billing-service, dst.) ke `kasku-internal` + `kasku-data` networks.

### `kasku-backend/infra/rabbitmq/definitions.json`
**Tidak perlu diubah** — setiap consumer service declare queue-nya sendiri saat startup (pattern yang benar untuk microservice).

---

## File Referensi Utama

| File | Relevansi |
|------|-----------|
| `auth-service/cmd/server/main.go` | DI pattern, graceful shutdown, health checker |
| `auth-service/configs/config.go` | Env var loading pattern (`requireEnv`, `getEnvOrDefault`) |
| `auth-service/internal/infrastructure/messaging/rabbitmq_publisher.go` | Exchange declaration, publish pattern |
| `auth-service/internal/infrastructure/persistence/db.go` | `RunMigrations`, `NewPostgresPool` |
| `auth-service/internal/usecase/register_usecase.go` | DB transaction + publish pattern |
| `api-gateway/proto/billing/v1/billing_grpc.go` | `protowire` encode/decode pattern — server side mirror ini |
| `api-gateway/internal/infrastructure/grpc/billing_client.go` | TierLimits struct + cache pattern |
| `api-gateway/internal/delivery/http/middleware/auth.go` | Header injection constants |
| `api-gateway/configs/config.go` | Service URL defaults, upstream config |
| `infra/postgres/00-init-databases.sh` | DB users + grants yang sudah ada |

---

## Verifikasi End-to-End

### Setelah Phase 1A (user-service + billing-service):
```bash
# 1. Start infra
docker compose up -d postgres rabbitmq redis

# 2. Start services
docker compose up -d auth-service user-service billing-service api-gateway

# 3. Register user
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","username":"testuser","password":"Test1234!"}'

# 4. Verify email (dari response register atau RabbitMQ message)
curl -X POST "http://localhost:8080/v1/auth/verify-email?token=<token>"

# 5. Cek tenant schema dibuat
psql kasku_finance -c "SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'tenant_%'"

# 6. Cek subscription FREE dibuat
psql kasku_billing -c "SELECT s.*, p.name FROM subscriptions s JOIN subscription_plans p ON s.plan_id=p.id WHERE s.user_id='<uuid>'"

# 7. Login → JWT harus punya tenant_schema + subscription_tier
curl -X POST http://localhost:8080/v1/auth/login \
  -d '{"email":"test@example.com","password":"Test1234!"}'
# Decode JWT payload: harus ada tenant_schema dan subscription_tier="FREE"

# 8. Cek api-gateway inject tier headers ke billing gRPC call
# (billingClient.GetTierLimits harus return limits dari DB, bukan fallback FREE)
```

### Setelah Phase 1B (finance-service + transaction-service):
```bash
# Login → dapat JWT
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/login -d '...' | jq -r .data.access_token)

# Buat akun
curl -X POST http://localhost:8080/v1/accounts \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"BCA","account_type":"BANK"}'

# Buat 4 akun → ke-4 harus return HTTP 402 (FREE limit = 3)
# ...

# Cek default categories (14 buah) sudah tersedia
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/categories
# harus return 14 items

# Buat 51 transaksi → ke-51 harus HTTP 402 (FREE limit = 50/bulan)
```

### Setelah Phase 1C (notification-service):
```bash
# Register user baru → check RabbitMQ Management UI (:15672)
# Queue kasku.notification-service harus kosong setelah beberapa detik
# Check email inbox (atau MailHog di dev)
```

### Unit tests:
```bash
cd kasku-backend/<service>
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
# Target: > 70% coverage untuk use case layer
```