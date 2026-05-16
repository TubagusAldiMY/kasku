# Phase 2 Implementation Plan — Investment, Price & Sync Services

## Goal

Implement the three Phase 2 microservices and integrate them into the existing KasKu SaaS stack:

1. **investment-service** (Go) — CRUD instrumen investasi + unit history + price aggregation
2. **price-service** (Rust) — Real-time price cache dari CoinGecko & metals.live
3. **sync-service** (Rust) — Offline sync engine, conflict resolution Server Wins

Plus required modifications to `api-gateway` and `docker-compose.yml`.

---

## User Review Required

> [!IMPORTANT]
> **Rust toolchain**: Phase 2 requires Rust (stable) installed on the system for building price-service and sync-service. Please confirm Rust is available or if we should add it to Docker-only builds.

> [!IMPORTANT]
> **Port assignments**: The plan.md has conflicting port assignments. The architecture doc says `investment: 8086/9086, price: 8087/9087, sync: 8088`, but plan.md says `investment: 8087/9086, price: 8088/9087, sync: 8089`. I'll use the **architecture doc ports** (8086/8087/8088) since notification-service already occupies 8089 in docker-compose.yml. Please confirm.

> [!WARNING]
> **notification-service port conflict**: notification-service is at port **8089** in docker-compose.yml, but the architecture doc lists it at 8089 as well. The sync-service plan.md entry says port 8089 which conflicts. I'll assign **sync-service → 8088** (HTTP only, no gRPC per plan).

## Open Questions

1. **CoinGecko API key**: `.env.example` has `COINGECKO_API_KEY=` — is a free tier (no key, demo mode) sufficient for dev or do you have a key?
2. **metals.live endpoint**: What is the exact URL? The PRD mentions `metals.live API` but no specific endpoint. Typically `https://api.metals.live/v1/spot/gold`.
3. **GOLD_USD_IDR_RATE**: Should this be a static env var or fetched from an API? Plan says env var.
4. **Rust project init**: Should we use `cargo init` directly or do you have a preferred Rust project template?

---

## Execution Order (Dependency-Driven)

```mermaid
graph TD
    A["Step 1: price-service (Rust)"] --> C["Step 3: investment-service (Go)"]
    B["Step 2: sync-service (Rust)"] --> D["Step 4: api-gateway modifications"]
    C --> D
    D --> E["Step 5: docker-compose + .env updates"]
    E --> F["Step 6: Verification"]
```

**Rationale**: investment-service depends on price-service gRPC for real-time prices. sync-service is independent. api-gateway modifications come last (adding routes + upstream URLs).

---

## Proposed Changes

### Component 1: price-service (Rust)

New Rust service using Axum + SQLx + Tonic + reqwest. HTTP port **8087**, gRPC port **9087**.
Database: `kasku_price`, DB user: `kasku_price_svc`.

#### [NEW] `kasku-backend/price-service/`

```
price-service/
├── Cargo.toml
├── Dockerfile
├── Makefile
├── build.rs                        # tonic-build for proto
├── migrations/
│   └── 001_create_price_cache.sql
├── proto/
│   └── price/v1/price.proto        # gRPC service definition
└── src/
    ├── main.rs                     # Axum + Tonic server bootstrap
    ├── config.rs                   # Env var loading (envy crate)
    ├── domain/
    │   ├── mod.rs
    │   ├── entity.rs               # PriceCache struct
    │   └── error.rs                # Domain errors
    ├── usecase/
    │   ├── mod.rs
    │   ├── get_price.rs            # GetPrice use case (cache check → API call → upsert)
    │   └── fetch_external.rs       # CoinGecko + metals.live HTTP clients
    ├── infrastructure/
    │   ├── mod.rs
    │   ├── db.rs                   # SQLx pool setup + migrations
    │   └── repository.rs           # PriceCacheRepository (SQLx impl)
    ├── delivery/
    │   ├── mod.rs
    │   ├── http_handler.rs         # GET /health, GET /v1/prices/:symbol
    │   └── grpc_handler.rs         # Tonic GetPrice service impl
    └── proto_gen/                  # Generated code from build.rs
```

**Key Dependencies (Cargo.toml)**:
- `axum 0.7`, `tokio 1`, `tonic 0.12`, `prost 0.13`
- `sqlx 0.8` (postgres, runtime-tokio)
- `reqwest 0.12` (json, rustls-tls)
- `serde 1`, `serde_json 1`
- `tracing 0.1`, `tracing-subscriber 0.3`
- `envy 0.4` (env var config)

**gRPC Proto** (`price.proto`):
```protobuf
syntax = "proto3";
package price.v1;

service PriceService {
  rpc GetPrice(GetPriceRequest) returns (GetPriceResponse);
  rpc GetPrices(GetPricesRequest) returns (GetPricesResponse);
}

message GetPriceRequest {
  string symbol = 1;
  string source = 2; // COINGECKO, METALS_LIVE, or empty for auto-detect
}
message GetPriceResponse {
  string symbol = 1;
  double price_idr = 2;
  double price_usd = 3;
  bool is_fresh = 4;
  string updated_at = 5; // RFC3339
}
message GetPricesRequest {
  repeated string symbols = 1;
}
message GetPricesResponse {
  repeated GetPriceResponse prices = 1;
}
```

**Use Case Logic**:
1. Check `price_cache` table: if `expires_at > NOW()` → return cached, `is_fresh=true`
2. If expired → call CoinGecko/metals.live with **5s timeout**
3. SSRF protection: whitelist domains `api.coingecko.com`, `api.metals.live`
4. On success → UPSERT into `price_cache`, return fresh data
5. On failure → return stale cached data with `is_fresh=false`

**Migration** (001):
```sql
-- Already defined in databaseScheme.md section 5.1
CREATE TABLE IF NOT EXISTS public.price_cache ( ... );
```

---

### Component 2: sync-service (Rust)

New Rust service using Axum + SQLx. HTTP port **8088**, no gRPC server.
Database: `kasku_finance` (tenant schemas), DB user: `kasku_sync_svc`.

#### [NEW] `kasku-backend/sync-service/`

```
sync-service/
├── Cargo.toml
├── Dockerfile
├── Makefile
└── src/
    ├── main.rs                     # Axum server bootstrap
    ├── config.rs                   # Env var loading
    ├── domain/
    │   ├── mod.rs
    │   ├── entity.rs               # SyncOperation, SyncResult, SyncLogEntry
    │   └── error.rs                # Domain errors
    ├── usecase/
    │   ├── mod.rs
    │   ├── push_sync.rs            # Process batch push operations
    │   └── pull_sync.rs            # Return changes since last_sync_timestamp
    ├── infrastructure/
    │   ├── mod.rs
    │   ├── db.rs                   # SQLx pool setup
    │   ├── tenant.rs               # Tenant schema validation (regex)
    │   └── repository.rs           # SyncRepository (SQLx impl)
    └── delivery/
        ├── mod.rs
        └── http_handler.rs         # POST /v1/sync/push, GET /v1/sync/pull, GET /health
```

**Key Dependencies (Cargo.toml)**:
- `axum 0.7`, `tokio 1`
- `sqlx 0.8` (postgres, runtime-tokio, uuid, chrono)
- `serde 1`, `serde_json 1`, `uuid 1`, `chrono 0.4`
- `tracing 0.1`, `tracing-subscriber 0.3`
- `envy 0.4`

**Endpoints**:
- `POST /v1/sync/push` — Batch push offline operations, returns processed count + conflicts
- `GET /v1/sync/pull?since={timestamp}` — Pull changes since timestamp
- `GET /health` — Health check (ping PostgreSQL)

**Push Sync Logic** (Server Wins):
1. Read `X-User-ID`, `X-Tenant-Schema` headers
2. Validate tenant schema via regex `^tenant_[0-9a-f_]{32,36}$`
3. For each operation in batch:
   - Check `sync_log` for duplicate `sync_id` (idempotency)
   - If entity exists and `updated_at > client_timestamp` → **SERVER_WINS** conflict
   - Otherwise execute operation (INSERT/UPDATE/soft DELETE)
   - Log to `{tenant}.sync_log`
4. Return `{processed: N, conflicts: [{entity_id, entity_type, server_data}]}`

**Pull Sync Logic**:
1. Query all entities where `updated_at > since_timestamp` across:
   - `{tenant}.financial_accounts`
   - `{tenant}.transactions`
   - `{tenant}.investment_assets`
2. Return delta changes + `server_timestamp`

---

### Component 3: investment-service (Go)

New Go service following finance-service patterns exactly. HTTP port **8086**, gRPC port **9086**.
Database: `kasku_finance` (tenant schemas), DB user: `kasku_investment_svc`.

#### [NEW] `kasku-backend/investment-service/`

```
investment-service/
├── cmd/server/main.go              # DI, graceful shutdown (mirror finance-service)
├── configs/config.go               # Env loading + PriceServiceGRPCAddr
├── Dockerfile
├── Makefile
├── go.mod
├── migrations/                     # Empty — tables created by provision_tenant()
└── internal/
    ├── domain/
    │   ├── entity/
    │   │   └── investment_asset.go  # InvestmentAsset, UnitHistory structs
    │   ├── errors/
    │   │   └── domain_errors.go     # ErrAssetNotFound, ErrAssetLimitReached, etc.
    │   └── repository/
    │       └── investment_repository.go  # Interface
    ├── delivery/
    │   ├── http/
    │   │   ├── handler/
    │   │   │   └── investment_handler.go  # HTTP handlers (mirror account_handler.go pattern)
    │   │   ├── middleware/
    │   │   │   └── correlation_id.go
    │   │   └── router.go
    │   └── grpc/                    # gRPC server (for sync-service Phase 2+)
    ├── infrastructure/
    │   ├── persistence/
    │   │   ├── db.go                # Pool + migrations
    │   │   └── postgres_investment_repository.go
    │   └── grpc/
    │       └── price_client.go      # gRPC client to price-service
    └── usecase/
        ├── create_asset_usecase.go
        ├── list_assets_usecase.go
        ├── get_asset_usecase.go
        ├── update_asset_usecase.go
        ├── delete_asset_usecase.go
        └── get_unit_history_usecase.go
```

**Domain Entity** (`investment_asset.go`):
```go
type AssetType string
const (
    AssetTypeCrypto     AssetType = "CRYPTO"
    AssetTypeGold       AssetType = "GOLD"
    AssetTypeStock      AssetType = "STOCK"
    AssetTypeMutualFund AssetType = "MUTUAL_FUND"
    AssetTypeOther      AssetType = "OTHER"
)

type InvestmentAsset struct {
    ID           uuid.UUID
    Name         string
    AssetType    AssetType
    Symbol       string
    Quantity     float64     // NUMERIC(30,10)
    AvgBuyPrice  float64     // NUMERIC(20,2)
    Currency     string
    IsDeleted    bool
    DeletedAt    *time.Time
    SortOrder    int
    CreatedAt    time.Time
    UpdatedAt    time.Time
    // Enriched fields (not persisted)
    CurrentPrice *float64   // from price-service
    IsPriceFresh *bool
}

type UnitHistory struct {
    ID              uuid.UUID
    AssetID         uuid.UUID
    TransactionType string   // BUY, SELL, ADJUST
    QuantityChange  float64
    PricePerUnit    float64
    TotalValue      float64  // generated
    Notes           string
    TransactionDate time.Time
    RecordedAt      time.Time
}
```

**Repository Interface**:
```go
type InvestmentAssetRepository interface {
    CountActive(ctx context.Context, tenantSchema string) (int, error)
    Create(ctx context.Context, tenantSchema string, asset *entity.InvestmentAsset) error
    List(ctx context.Context, tenantSchema string) ([]entity.InvestmentAsset, error)
    GetByID(ctx context.Context, tenantSchema, id string) (*entity.InvestmentAsset, error)
    Update(ctx context.Context, tenantSchema string, asset *entity.InvestmentAsset) error
    SoftDelete(ctx context.Context, tenantSchema, id string) error
    // Unit History
    CreateUnitHistory(ctx context.Context, tenantSchema string, entry *entity.UnitHistory) error
    GetUnitHistory(ctx context.Context, tenantSchema, assetID string, limitMonths int) ([]entity.UnitHistory, error)
}
```

**HTTP Endpoints** (all require `X-User-ID`, `X-Tenant-Schema`):
| Method | Path | Handler | Tier Check |
|--------|------|---------|------------|
| GET | /health | Health | — |
| GET | /v1/investments | ListAssets | — |
| POST | /v1/investments | CreateAsset | `X-Tier-Max-Investments` |
| GET | /v1/investments/:id | GetAsset | — |
| PUT | /v1/investments/:id | UpdateAsset | — |
| DELETE | /v1/investments/:id | DeleteAsset | — |
| GET | /v1/investments/:id/history | GetUnitHistory | `X-Tier-History-Months` |

**Price Client** (`price_client.go`):
- gRPC client to price-service at `PRICE_GRPC_ADDR` (default: `price-service:9087`)
- Called during `ListAssets` and `GetAsset` to enrich response with current price
- Timeout: 300ms, fallback: return asset without price data

**Config additions**:
```go
type PriceConfig struct {
    GRPCAddr string // PRICE_GRPC_ADDR, default "price-service:9087"
    Timeout  time.Duration
}
```

---

### Component 4: api-gateway Modifications

#### [MODIFY] [config.go](file:///home/tubsamy/Tubsamy-dev/Projects/kasku/kasku-backend/api-gateway/configs/config.go)

Add upstream URLs for 3 new services:
```diff
 type ProxyConfig struct {
     AuthServiceURL        string
     UserServiceURL        string
     BillingServiceURL     string
     FinanceServiceURL     string
     TransactionServiceURL string
+    InvestmentServiceURL  string
+    SyncServiceURL        string
 }
```

Add loading:
```diff
 cfg.Proxy.TransactionServiceURL = getEnvOrDefault("TRANSACTION_SERVICE_URL", "http://transaction-service:8085")
+cfg.Proxy.InvestmentServiceURL = getEnvOrDefault("INVESTMENT_SERVICE_URL", "http://investment-service:8086")
+cfg.Proxy.SyncServiceURL = getEnvOrDefault("SYNC_SERVICE_URL", "http://sync-service:8088")
```

> [!NOTE]
> price-service does NOT need a gateway route — it's only called internally by investment-service via gRPC.

#### [MODIFY] [main.go](file:///home/tubsamy/Tubsamy-dev/Projects/kasku/kasku-backend/api-gateway/cmd/server/main.go)

Add upstream entries:
```diff
 upstreams := map[string]string{
     "auth":        cfg.Proxy.AuthServiceURL,
     "user":        cfg.Proxy.UserServiceURL,
     "billing":     cfg.Proxy.BillingServiceURL,
     "finance":     cfg.Proxy.FinanceServiceURL,
     "transaction": cfg.Proxy.TransactionServiceURL,
+    "investment":  cfg.Proxy.InvestmentServiceURL,
+    "sync":        cfg.Proxy.SyncServiceURL,
 }
```

#### [MODIFY] [router.go](file:///home/tubsamy/Tubsamy-dev/Projects/kasku/kasku-backend/api-gateway/internal/delivery/http/router.go)

Add route groups after the categories block:
```go
// ── /v1/investments/** ────────────────────────────────────────────────
v1Investments := r.Group("/v1/investments")
v1Investments.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
{
    v1Investments.Any("/*path", cfg.ProxyHandler.ProxyTo("investment"))
}

// ── /v1/sync/** ───────────────────────────────────────────────────────
v1Sync := r.Group("/v1/sync")
v1Sync.Use(cfg.AuthMiddleware, cfg.RateLimitMiddleware)
{
    v1Sync.Any("/*path", cfg.ProxyHandler.ProxyTo("sync"))
}
```

---

### Component 5: docker-compose.yml + .env Updates

#### [MODIFY] [docker-compose.yml](file:///home/tubsamy/Tubsamy-dev/Projects/kasku/kasku-backend/docker-compose.yml)

Add 3 service blocks after notification-service:

```yaml
  # ============================================================
  # investment-service
  # ============================================================
  investment-service:
    build: ./investment-service
    container_name: kasku-investment-service
    restart: unless-stopped
    env_file:
      - .env
    environment:
      SERVER_PORT: "8086"
      GRPC_PORT: "9086"
      APP_ENV: ${APP_ENV:-production}
      LOG_LEVEL: ${LOG_LEVEL:-info}
      POSTGRES_DSN: postgres://kasku_investment_svc:${KASKU_INVESTMENT_DB_PASS}@postgres:5432/kasku_finance?sslmode=disable
      PRICE_GRPC_ADDR: kasku-price-service:9087
    networks:
      - kasku-internal
      - kasku-data
    depends_on:
      postgres:
        condition: service_healthy
      user-service:
        condition: service_healthy
      price-service:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8086/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 10s
    stop_grace_period: 35s

  # ============================================================
  # price-service (Rust)
  # ============================================================
  price-service:
    build: ./price-service
    container_name: kasku-price-service
    restart: unless-stopped
    env_file:
      - .env
    environment:
      HTTP_PORT: "8087"
      GRPC_PORT: "9087"
      APP_ENV: ${APP_ENV:-production}
      LOG_LEVEL: ${LOG_LEVEL:-info}
      DATABASE_URL: postgres://kasku_price_svc:${KASKU_PRICE_DB_PASS}@postgres:5432/kasku_price?sslmode=disable
      COINGECKO_API_KEY: ${COINGECKO_API_KEY:-}
      PRICE_CACHE_TTL_SECONDS: ${PRICE_CACHE_TTL_SECONDS:-900}
      EXTERNAL_REQUEST_TIMEOUT_SECONDS: ${EXTERNAL_REQUEST_TIMEOUT_SECONDS:-10}
    networks:
      - kasku-internal
      - kasku-data
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8087/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 15s
    stop_grace_period: 35s

  # ============================================================
  # sync-service (Rust)
  # ============================================================
  sync-service:
    build: ./sync-service
    container_name: kasku-sync-service
    restart: unless-stopped
    env_file:
      - .env
    environment:
      HTTP_PORT: "8088"
      APP_ENV: ${APP_ENV:-production}
      LOG_LEVEL: ${LOG_LEVEL:-info}
      DATABASE_URL: postgres://kasku_sync_svc:${KASKU_SYNC_DB_PASS}@postgres:5432/kasku_finance?sslmode=disable
    networks:
      - kasku-internal
      - kasku-data
    depends_on:
      postgres:
        condition: service_healthy
      user-service:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8088/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 15s
    stop_grace_period: 35s
```

Also add to api-gateway environment:
```diff
      TRANSACTION_SERVICE_URL: http://kasku-transaction-service:8085
+     INVESTMENT_SERVICE_URL: http://kasku-investment-service:8086
+     SYNC_SERVICE_URL: http://kasku-sync-service:8088
```

#### [MODIFY] [.env.example](file:///home/tubsamy/Tubsamy-dev/Projects/kasku/kasku-backend/.env.example)

Add `GOLD_USD_IDR_RATE` and `METALS_LIVE_URL` entries.

#### [MODIFY] [.env](file:///home/tubsamy/Tubsamy-dev/Projects/kasku/kasku-backend/.env)

Add matching values for new variables.

---

## Verification Plan

### Automated Tests

```bash
# 1. Build all services
cd kasku-backend
docker compose build price-service sync-service investment-service api-gateway

# 2. Start infra + all services
docker compose up -d

# 3. Verify health endpoints
curl http://localhost:8087/health  # price-service
curl http://localhost:8088/health  # sync-service
curl http://localhost:8086/health  # investment-service

# 4. Test price-service
# Get Bitcoin price (CoinGecko)
curl http://localhost:8087/v1/prices/bitcoin

# 5. Test investment-service (requires JWT)
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"testuser","password":"Test1234!"}' | jq -r .data.access_token)

# Create instrument
curl -X POST http://localhost:8080/v1/investments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Bitcoin","asset_type":"CRYPTO","symbol":"BTC","quantity":0.5,"avg_buy_price":450000000}'

# List instruments (should include current price from price-service)
curl http://localhost:8080/v1/investments \
  -H "Authorization: Bearer $TOKEN"

# Create 3rd instrument → should return HTTP 402 for FREE tier (limit=2)
# ...

# 6. Test sync-service
curl -X POST http://localhost:8080/v1/sync/push \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"operations":[{"sync_id":"test-uuid","entity_type":"financial_account","operation":"create","payload":{"name":"Test","account_type":"CASH"},"client_timestamp":"2026-05-10T00:00:00Z"}]}'

curl "http://localhost:8080/v1/sync/pull?since=2026-05-01T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN"

# 7. Unit tests per service
cd kasku-backend/investment-service && go test ./... -race -cover
cd kasku-backend/price-service && cargo test
cd kasku-backend/sync-service && cargo test
```

### Manual Verification
- Verify price-service returns `is_fresh: false` when CoinGecko is unreachable
- Verify investment tier limit (FREE = 2 instruments) returns HTTP 402
- Verify sync conflict resolution returns server data when conflict detected
- Check RabbitMQ Management UI — no new queues needed (these services don't consume events)
