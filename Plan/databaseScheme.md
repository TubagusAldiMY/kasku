# KasKu — Database Schema
# Multi-Tenant SaaS: Schema-per-Tenant Architecture

**Versi:** 2.0.0 (Pivot: Single-DB Monolith → Multi-DB Microservices + Schema-per-Tenant)
**Tanggal:** 2026-04-27
**Owner:** TubsAMY — admin@tubsamy.tech

---

## Daftar Isi

1. [Database Architecture Overview](#database-architecture-overview)
2. [DDL: kasku_auth](#ddl-kasku_auth)
3. [DDL: kasku_billing](#ddl-kasku_billing)
4. [DDL: kasku_finance (Schema per Tenant)](#ddl-kasku_finance)
5. [DDL: kasku_price](#ddl-kasku_price)
6. [DDL: kasku_admin](#ddl-kasku_admin)
7. [Migration Files Structure](#migration-files-structure)
8. [Tenant Schema Provisioning Function](#tenant-schema-provisioning-function)
9. [Index Strategy](#index-strategy)
10. [Database Security — Least Privilege](#database-security)
11. [Tier Limit Enforcement Pattern](#tier-limit-enforcement-pattern)

---

## 1. Database Architecture Overview {#database-architecture-overview}

Satu PostgreSQL instance menjalankan 5 database. Setiap database dimiliki oleh satu atau
lebih service. Service hanya boleh mengakses database miliknya sendiri.

```
PostgreSQL Instance (postgres:5432)
│
├── kasku_auth          → auth-service (owner)
│   └── public schema
│       ├── users
│       ├── refresh_tokens
│       └── email_verifications
│
├── kasku_billing       → billing-service (owner)
│   └── public schema
│       ├── subscription_plans  [+ seed data]
│       ├── subscriptions
│       └── payments
│
├── kasku_finance       → finance-service, transaction-service,
│   │                     investment-service, sync-service (shared ownership)
│   │                     user-service (DDL provisioning only)
│   │
│   ├── public schema   [kosong, tidak digunakan untuk data]
│   │
│   └── tenant_{uuid}   [satu schema per user]
│       ├── financial_accounts
│       ├── balance_history
│       ├── investment_assets
│       ├── unit_history
│       ├── transactions
│       ├── categories  [+ seed data]
│       └── sync_log
│
├── kasku_price         → price-service (owner)
│   └── public schema
│       └── price_cache
│
└── kasku_admin         → admin-service (owner)
    └── public schema
        └── admin_users
```

**Mapping Service → Database:**

| Service | Database(s) | Akses |
|---------|-------------|-------|
| auth-service | kasku_auth | Read/Write |
| user-service | kasku_finance (DDL), kasku_billing (INSERT subscription) | DDL + Limited Write |
| billing-service | kasku_billing | Read/Write |
| finance-service | kasku_finance | Read/Write (tenant schema) |
| transaction-service | kasku_finance | Read/Write (tenant schema) |
| investment-service | kasku_finance | Read/Write (tenant schema) |
| sync-service | kasku_finance | Read/Write (tenant schema) |
| price-service | kasku_price | Read/Write |
| notification-service | — (stateless, event-driven) | — |
| admin-service | kasku_admin (R/W), kasku_auth (R), kasku_billing (R) | Mixed |
| api-gateway | — (Redis for rate limit/blacklist) | — |

---

## 2. DDL: kasku_auth {#ddl-kasku_auth}

Database milik `auth-service`. Menyimpan credentials, session tokens, dan email verifications.

### 2.1 Tabel `public.users`

```sql
CREATE TABLE IF NOT EXISTS public.users (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    email               VARCHAR(254)    NOT NULL,
    username            VARCHAR(30)     NOT NULL,
    password_hash       TEXT            NOT NULL,
    is_active           BOOLEAN         NOT NULL DEFAULT false,
    email_verified      BOOLEAN         NOT NULL DEFAULT false,
    failed_login_count  SMALLINT        NOT NULL DEFAULT 0
                            CHECK (failed_login_count >= 0 AND failed_login_count <= 10),
    locked_until        TIMESTAMPTZ     NULL,
    last_login_at       TIMESTAMPTZ     NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
    ON public.users (LOWER(email));

CREATE UNIQUE INDEX IF NOT EXISTS users_username_unique_idx
    ON public.users (LOWER(username));

CREATE INDEX IF NOT EXISTS users_is_active_idx
    ON public.users (is_active)
    WHERE is_active = true;

COMMENT ON TABLE public.users IS 'KasKu user accounts. One account = one tenant.';
COMMENT ON COLUMN public.users.email IS 'Case-insensitive unique. Stored as provided, compared lowercase.';
COMMENT ON COLUMN public.users.password_hash IS 'Argon2id hash. Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>';
COMMENT ON COLUMN public.users.failed_login_count IS 'Resets to 0 on successful login.';
COMMENT ON COLUMN public.users.locked_until IS 'NULL means not locked. Account locked if locked_until > now().';
```

### 2.2 Tabel `public.refresh_tokens`

```sql
CREATE TABLE IF NOT EXISTS public.refresh_tokens (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID            NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)        NOT NULL,
    user_agent  VARCHAR(512)    NULL,
    ip_address  INET            NULL,
    expires_at  TIMESTAMPTZ     NOT NULL,
    is_revoked  BOOLEAN         NOT NULL DEFAULT false,
    revoked_at  TIMESTAMPTZ     NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS refresh_tokens_token_hash_unique_idx
    ON public.refresh_tokens (token_hash);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_active_idx
    ON public.refresh_tokens (user_id)
    WHERE is_revoked = false;

CREATE INDEX IF NOT EXISTS refresh_tokens_expires_at_idx
    ON public.refresh_tokens (expires_at)
    WHERE is_revoked = false;

COMMENT ON TABLE public.refresh_tokens IS 'SHA-256 hashed refresh tokens. Raw token stored in HttpOnly cookie only.';
COMMENT ON COLUMN public.refresh_tokens.token_hash IS 'SHA-256(raw_token) as lowercase hex. 32 bytes = 64 hex chars.';
COMMENT ON COLUMN public.refresh_tokens.ip_address IS 'Client IP at time of token issuance. For security audit only.';
```

### 2.3 Tabel `public.email_verifications`

```sql
CREATE TABLE IF NOT EXISTS public.email_verifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)    NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS email_verifications_token_hash_idx
    ON public.email_verifications (token_hash);

CREATE INDEX IF NOT EXISTS email_verifications_user_id_unverified_idx
    ON public.email_verifications (user_id)
    WHERE verified_at IS NULL;

COMMENT ON TABLE public.email_verifications IS 'Email verification tokens. Expires in 24 hours after creation.';
COMMENT ON COLUMN public.email_verifications.token_hash IS 'SHA-256(raw_token). Raw token sent to user email only.';
```

### 2.4 Password Reset Tokens

```sql
CREATE TABLE IF NOT EXISTS public.password_reset_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)    NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS password_reset_tokens_token_hash_idx
    ON public.password_reset_tokens (token_hash);

CREATE INDEX IF NOT EXISTS password_reset_tokens_user_id_unused_idx
    ON public.password_reset_tokens (user_id)
    WHERE used_at IS NULL;

COMMENT ON TABLE public.password_reset_tokens IS 'One-time password reset tokens. Expires in 1 hour.';
```

### 2.5 Trigger `updated_at` untuk tabel `users`

```sql
CREATE OR REPLACE FUNCTION public.set_updated_at_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON public.users
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
```

---

## 3. DDL: kasku_billing {#ddl-kasku_billing}

Database milik `billing-service`. Menyimpan subscription plans, active subscriptions,
dan payment records.

### 3.1 Tabel `public.subscription_plans`

```sql
CREATE TABLE IF NOT EXISTS public.subscription_plans (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(20) NOT NULL CHECK (name IN ('FREE', 'BASIC', 'PRO')),
    display_name    VARCHAR(50) NOT NULL,
    price_idr       INTEGER     NOT NULL CHECK (price_idr >= 0),
    billing_cycle   VARCHAR(10) NOT NULL DEFAULT 'MONTHLY' CHECK (billing_cycle IN ('MONTHLY', 'YEARLY', 'NONE')),
    limits_json     JSONB       NOT NULL,
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS subscription_plans_name_unique_idx
    ON public.subscription_plans (name);

CREATE INDEX IF NOT EXISTS subscription_plans_active_idx
    ON public.subscription_plans (is_active)
    WHERE is_active = true;

COMMENT ON COLUMN public.subscription_plans.limits_json IS
    'JSON object defining tier limits. See tier_limit_enforcement section for schema.';
COMMENT ON COLUMN public.subscription_plans.price_idr IS
    'Price in Indonesian Rupiah cents (integer). 29000 = Rp 29.000.';
```

### 3.2 Seed Data `subscription_plans`

```sql
INSERT INTO public.subscription_plans (id, name, display_name, price_idr, billing_cycle, limits_json, is_active)
VALUES
(
    '00000000-0000-0000-0000-000000000001',
    'FREE',
    'Gratis',
    0,
    'NONE',
    '{
        "max_transactions_per_month": 50,
        "max_financial_accounts": 3,
        "max_investment_instruments": 2,
        "history_retention_months": 3,
        "email_notifications_enabled": false,
        "export_csv_enabled": false,
        "offline_sync_enabled": true,
        "realtime_price_enabled": true
    }'::jsonb,
    true
),
(
    '00000000-0000-0000-0000-000000000002',
    'BASIC',
    'Basic',
    29000,
    'MONTHLY',
    '{
        "max_transactions_per_month": 500,
        "max_financial_accounts": 10,
        "max_investment_instruments": 10,
        "history_retention_months": 12,
        "email_notifications_enabled": true,
        "export_csv_enabled": false,
        "offline_sync_enabled": true,
        "realtime_price_enabled": true
    }'::jsonb,
    true
),
(
    '00000000-0000-0000-0000-000000000003',
    'PRO',
    'Pro',
    59000,
    'MONTHLY',
    '{
        "max_transactions_per_month": -1,
        "max_financial_accounts": -1,
        "max_investment_instruments": -1,
        "history_retention_months": -1,
        "email_notifications_enabled": true,
        "export_csv_enabled": true,
        "offline_sync_enabled": true,
        "realtime_price_enabled": true
    }'::jsonb,
    true
)
ON CONFLICT (name) DO UPDATE SET
    display_name    = EXCLUDED.display_name,
    price_idr       = EXCLUDED.price_idr,
    limits_json     = EXCLUDED.limits_json,
    updated_at      = now();

COMMENT ON TABLE public.subscription_plans IS
    'Subscription plan catalog. limits_json value -1 means unlimited.';
```

### 3.3 Tabel `public.subscriptions`

```sql
CREATE TYPE subscription_status AS ENUM (
    'TRIALING',
    'ACTIVE',
    'PAST_DUE',
    'CANCELLED',
    'EXPIRED'
);

CREATE TABLE IF NOT EXISTS public.subscriptions (
    id                      UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID                NOT NULL,
    plan_id                 UUID                NOT NULL REFERENCES public.subscription_plans(id),
    status                  subscription_status NOT NULL DEFAULT 'ACTIVE',
    current_period_start    TIMESTAMPTZ         NOT NULL DEFAULT now(),
    current_period_end      TIMESTAMPTZ         NULL,
    cancelled_at            TIMESTAMPTZ         NULL,
    cancel_at_period_end    BOOLEAN             NOT NULL DEFAULT false,
    created_at              TIMESTAMPTZ         NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ         NOT NULL DEFAULT now(),

    CONSTRAINT subscriptions_cancel_consistency_check
        CHECK (
            (status = 'CANCELLED' AND cancelled_at IS NOT NULL)
            OR (status != 'CANCELLED' AND cancelled_at IS NULL)
        )
);

CREATE UNIQUE INDEX IF NOT EXISTS subscriptions_user_active_unique_idx
    ON public.subscriptions (user_id)
    WHERE status IN ('ACTIVE', 'TRIALING', 'PAST_DUE');

CREATE INDEX IF NOT EXISTS subscriptions_user_id_idx
    ON public.subscriptions (user_id);

CREATE INDEX IF NOT EXISTS subscriptions_status_period_end_idx
    ON public.subscriptions (status, current_period_end)
    WHERE status IN ('ACTIVE', 'TRIALING') AND current_period_end IS NOT NULL;

CREATE TRIGGER subscriptions_set_updated_at
    BEFORE UPDATE ON public.subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();

COMMENT ON COLUMN public.subscriptions.user_id IS
    'References auth-service users.id. No FK across databases — enforced at application layer.';
COMMENT ON COLUMN public.subscriptions.current_period_end IS
    'NULL for FREE plan (no expiry). Non-null for BASIC and PRO.';
COMMENT ON COLUMN public.subscriptions.cancel_at_period_end IS
    'True when user requested cancel but subscription still active until period_end.';
```

### 3.4 Tabel `public.payments`

```sql
CREATE TYPE payment_status AS ENUM (
    'PENDING',
    'SETTLEMENT',
    'EXPIRE',
    'CANCEL',
    'DENY',
    'REFUND',
    'FAILURE'
);

CREATE TABLE IF NOT EXISTS public.payments (
    id                          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                     UUID            NOT NULL,
    subscription_id             UUID            NULL REFERENCES public.subscriptions(id) ON DELETE SET NULL,
    midtrans_order_id           VARCHAR(100)    NOT NULL,
    midtrans_transaction_id     VARCHAR(100)    NULL,
    amount_idr                  INTEGER         NOT NULL CHECK (amount_idr > 0),
    status                      payment_status  NOT NULL DEFAULT 'PENDING',
    payment_method              VARCHAR(50)     NULL,
    snap_token                  VARCHAR(255)    NULL,
    paid_at                     TIMESTAMPTZ     NULL,
    expired_at                  TIMESTAMPTZ     NULL,
    created_at                  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ     NOT NULL DEFAULT now(),

    CONSTRAINT payments_order_id_format_check
        CHECK (midtrans_order_id ~ '^KASKU-[0-9a-f]{8}-[0-9]+$'),
    CONSTRAINT payments_settlement_requires_paid_at_check
        CHECK (
            (status = 'SETTLEMENT' AND paid_at IS NOT NULL)
            OR (status != 'SETTLEMENT')
        )
);

CREATE UNIQUE INDEX IF NOT EXISTS payments_midtrans_order_id_unique_idx
    ON public.payments (midtrans_order_id);

CREATE INDEX IF NOT EXISTS payments_user_id_status_idx
    ON public.payments (user_id, status);

CREATE INDEX IF NOT EXISTS payments_midtrans_transaction_id_idx
    ON public.payments (midtrans_transaction_id)
    WHERE midtrans_transaction_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS payments_created_at_idx
    ON public.payments (created_at DESC);

CREATE TRIGGER payments_set_updated_at
    BEFORE UPDATE ON public.payments
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();

COMMENT ON TABLE public.payments IS 'Payment records from Midtrans. One record per payment attempt.';
COMMENT ON COLUMN public.payments.midtrans_order_id IS
    'Format: KASKU-{first_8_chars_user_id}-{unix_timestamp}. Unique per payment attempt.';
COMMENT ON COLUMN public.payments.snap_token IS
    'Midtrans Snap token for frontend redirect. Cleared after payment completes.';
```

**Payment Status FSM:**
```
PENDING → SETTLEMENT  (Midtrans notification: transaction_status=settlement)
PENDING → EXPIRE      (Midtrans notification: transaction_status=expire, atau TTL 24h)
PENDING → CANCEL      (Midtrans notification: transaction_status=cancel)
PENDING → DENY        (Midtrans notification: transaction_status=deny)
SETTLEMENT → REFUND   (Manual via admin-service, exceptional case)
```

---

## 4. DDL: kasku_finance (Schema per Tenant) {#ddl-kasku_finance}

Database ini diakses oleh finance-service, transaction-service, investment-service,
sync-service, dan user-service (hanya untuk DDL provisioning).

**Setiap user memiliki satu schema** dengan nama `tenant_{user_uuid_sanitized}`.
Contoh: user dengan UUID `550e8400-e29b-41d4-a716-446655440000` mendapat schema
`tenant_550e8400_e29b_41d4_a716_446655440000`.

DDL di bawah ini adalah template yang dijalankan saat provisioning setiap tenant baru.
Placeholder `{SCHEMA}` digantikan dengan nama schema aktual.

### 4.1 Tabel `{SCHEMA}.financial_accounts`

```sql
CREATE TABLE IF NOT EXISTS {SCHEMA}.financial_accounts (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100)    NOT NULL,
    account_type    VARCHAR(20)     NOT NULL
                        CHECK (account_type IN ('BANK', 'EWALLET', 'CASH', 'INVESTMENT')),
    currency        CHAR(3)         NOT NULL DEFAULT 'IDR',
    balance         NUMERIC(20, 2)  NOT NULL DEFAULT 0.00,
    institution     VARCHAR(100)    NULL,
    account_number  VARCHAR(50)     NULL,
    color           CHAR(7)         NULL CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    icon            VARCHAR(50)     NULL,
    is_active       BOOLEAN         NOT NULL DEFAULT true,
    is_deleted      BOOLEAN         NOT NULL DEFAULT false,
    deleted_at      TIMESTAMPTZ     NULL,
    sort_order      SMALLINT        NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),

    CONSTRAINT financial_accounts_delete_consistency_check
        CHECK (
            (is_deleted = true AND deleted_at IS NOT NULL)
            OR (is_deleted = false AND deleted_at IS NULL)
        )
);

CREATE INDEX IF NOT EXISTS financial_accounts_active_idx
    ON {SCHEMA}.financial_accounts (is_deleted, is_active)
    WHERE is_deleted = false;

CREATE TRIGGER financial_accounts_set_updated_at
    BEFORE UPDATE ON {SCHEMA}.financial_accounts
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
```

### 4.2 Tabel `{SCHEMA}.balance_history`

```sql
CREATE TABLE IF NOT EXISTS {SCHEMA}.balance_history (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID            NOT NULL REFERENCES {SCHEMA}.financial_accounts(id)
                        ON DELETE CASCADE,
    balance_before  NUMERIC(20, 2)  NOT NULL,
    balance_after   NUMERIC(20, 2)  NOT NULL,
    change_amount   NUMERIC(20, 2)  NOT NULL
                        GENERATED ALWAYS AS (balance_after - balance_before) STORED,
    change_reason   VARCHAR(50)     NOT NULL
                        CHECK (change_reason IN ('TRANSACTION', 'MANUAL_ADJUST', 'SYNC', 'INITIAL')),
    reference_id    UUID            NULL,
    recorded_at     TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS balance_history_account_id_recorded_at_idx
    ON {SCHEMA}.balance_history (account_id, recorded_at DESC);

COMMENT ON TABLE {SCHEMA}.balance_history IS
    'Append-only audit log of balance changes. Never UPDATE or DELETE records.';
COMMENT ON COLUMN {SCHEMA}.balance_history.reference_id IS
    'Optional: transaction_id or sync_log_id that caused this balance change.';
```

### 4.3 Tabel `{SCHEMA}.categories`

```sql
CREATE TABLE IF NOT EXISTS {SCHEMA}.categories (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(50) NOT NULL,
    type        VARCHAR(10) NOT NULL CHECK (type IN ('INCOME', 'EXPENSE', 'TRANSFER')),
    icon        VARCHAR(50) NULL,
    color       CHAR(7)     NULL CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    is_default  BOOLEAN     NOT NULL DEFAULT false,
    is_deleted  BOOLEAN     NOT NULL DEFAULT false,
    deleted_at  TIMESTAMPTZ NULL,
    sort_order  SMALLINT    NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS categories_type_active_idx
    ON {SCHEMA}.categories (type, is_deleted)
    WHERE is_deleted = false;

CREATE TRIGGER categories_set_updated_at
    BEFORE UPDATE ON {SCHEMA}.categories
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
```

### 4.4 Seed Categories Default

```sql
INSERT INTO {SCHEMA}.categories (name, type, icon, color, is_default, sort_order)
VALUES
    -- EXPENSE categories
    ('Makanan & Minuman',   'EXPENSE', 'utensils',      '#F97316', true, 1),
    ('Transport',           'EXPENSE', 'car',           '#3B82F6', true, 2),
    ('Belanja',             'EXPENSE', 'shopping-bag',  '#8B5CF6', true, 3),
    ('Tagihan & Utilitas',  'EXPENSE', 'zap',           '#EF4444', true, 4),
    ('Kesehatan',           'EXPENSE', 'heart',         '#EC4899', true, 5),
    ('Hiburan',             'EXPENSE', 'music',         '#F59E0B', true, 6),
    ('Pendidikan',          'EXPENSE', 'book',          '#06B6D4', true, 7),
    ('Lain-lain',           'EXPENSE', 'more-horizontal','#6B7280', true, 8),
    -- INCOME categories
    ('Gaji',                'INCOME',  'briefcase',     '#10B981', true, 9),
    ('Freelance',           'INCOME',  'laptop',        '#34D399', true, 10),
    ('Investasi',           'INCOME',  'trending-up',   '#6EE7B7', true, 11),
    ('Lain-lain',           'INCOME',  'plus-circle',   '#A7F3D0', true, 12),
    -- TRANSFER categories
    ('Transfer Masuk',      'TRANSFER','arrow-down-left','#60A5FA', true, 13),
    ('Transfer Keluar',     'TRANSFER','arrow-up-right', '#93C5FD', true, 14);
```

### 4.5 Tabel `{SCHEMA}.transactions`

```sql
CREATE TABLE IF NOT EXISTS {SCHEMA}.transactions (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    sync_id         UUID            NULL,
    account_id      UUID            NOT NULL REFERENCES {SCHEMA}.financial_accounts(id)
                        ON DELETE RESTRICT,
    category_id     UUID            NOT NULL REFERENCES {SCHEMA}.categories(id)
                        ON DELETE RESTRICT,
    transaction_type VARCHAR(10)    NOT NULL CHECK (transaction_type IN ('INCOME', 'EXPENSE', 'TRANSFER')),
    amount          NUMERIC(20, 2)  NOT NULL CHECK (amount > 0),
    description     VARCHAR(500)    NULL,
    notes           TEXT            NULL,
    transaction_date DATE            NOT NULL,
    to_account_id   UUID            NULL REFERENCES {SCHEMA}.financial_accounts(id)
                        ON DELETE RESTRICT,
    is_deleted      BOOLEAN         NOT NULL DEFAULT false,
    deleted_at      TIMESTAMPTZ     NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),

    CONSTRAINT transactions_transfer_requires_to_account_check
        CHECK (
            (transaction_type = 'TRANSFER' AND to_account_id IS NOT NULL)
            OR (transaction_type != 'TRANSFER' AND to_account_id IS NULL)
        ),
    CONSTRAINT transactions_delete_consistency_check
        CHECK (
            (is_deleted = true AND deleted_at IS NOT NULL)
            OR (is_deleted = false AND deleted_at IS NULL)
        )
);

-- Partial unique index: sync_id unik di antara record yang tidak dihapus
CREATE UNIQUE INDEX IF NOT EXISTS transactions_sync_id_unique_idx
    ON {SCHEMA}.transactions (sync_id)
    WHERE sync_id IS NOT NULL AND is_deleted = false;

CREATE INDEX IF NOT EXISTS transactions_account_date_idx
    ON {SCHEMA}.transactions (account_id, transaction_date DESC)
    WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS transactions_category_date_idx
    ON {SCHEMA}.transactions (category_id, transaction_date DESC)
    WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS transactions_date_type_idx
    ON {SCHEMA}.transactions (transaction_date DESC, transaction_type)
    WHERE is_deleted = false;

CREATE TRIGGER transactions_set_updated_at
    BEFORE UPDATE ON {SCHEMA}.transactions
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();

COMMENT ON COLUMN {SCHEMA}.transactions.sync_id IS
    'Client-generated UUID for offline sync idempotency. Null for server-created records.';
COMMENT ON COLUMN {SCHEMA}.transactions.to_account_id IS
    'Only used for TRANSFER type transactions.';
```

### 4.6 Tabel `{SCHEMA}.investment_assets`

```sql
CREATE TABLE IF NOT EXISTS {SCHEMA}.investment_assets (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100)    NOT NULL,
    asset_type      VARCHAR(20)     NOT NULL
                        CHECK (asset_type IN ('CRYPTO', 'GOLD', 'STOCK', 'MUTUAL_FUND', 'OTHER')),
    symbol          VARCHAR(20)     NOT NULL,
    quantity        NUMERIC(30, 10) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    avg_buy_price   NUMERIC(20, 2)  NOT NULL DEFAULT 0 CHECK (avg_buy_price >= 0),
    currency        CHAR(3)         NOT NULL DEFAULT 'IDR',
    is_deleted      BOOLEAN         NOT NULL DEFAULT false,
    deleted_at      TIMESTAMPTZ     NULL,
    sort_order      SMALLINT        NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS investment_assets_active_idx
    ON {SCHEMA}.investment_assets (is_deleted)
    WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS investment_assets_symbol_type_idx
    ON {SCHEMA}.investment_assets (symbol, asset_type)
    WHERE is_deleted = false;

CREATE TRIGGER investment_assets_set_updated_at
    BEFORE UPDATE ON {SCHEMA}.investment_assets
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
```

### 4.7 Tabel `{SCHEMA}.unit_history`

```sql
CREATE TABLE IF NOT EXISTS {SCHEMA}.unit_history (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id            UUID            NOT NULL REFERENCES {SCHEMA}.investment_assets(id)
                            ON DELETE CASCADE,
    transaction_type    VARCHAR(10)     NOT NULL CHECK (transaction_type IN ('BUY', 'SELL', 'ADJUST')),
    quantity_change     NUMERIC(30, 10) NOT NULL,
    price_per_unit      NUMERIC(20, 2)  NOT NULL CHECK (price_per_unit > 0),
    total_value         NUMERIC(20, 2)  NOT NULL
                            GENERATED ALWAYS AS (ABS(quantity_change) * price_per_unit) STORED,
    notes               VARCHAR(500)    NULL,
    transaction_date    DATE            NOT NULL,
    recorded_at         TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS unit_history_asset_id_date_idx
    ON {SCHEMA}.unit_history (asset_id, transaction_date DESC);

COMMENT ON TABLE {SCHEMA}.unit_history IS
    'Append-only audit log of investment unit changes. Never UPDATE or DELETE records.';
COMMENT ON COLUMN {SCHEMA}.unit_history.quantity_change IS
    'Positive for BUY/ADJUST increase, negative for SELL/ADJUST decrease.';
```

### 4.8 Tabel `{SCHEMA}.sync_log`

```sql
CREATE TABLE IF NOT EXISTS {SCHEMA}.sync_log (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    operation       VARCHAR(20) NOT NULL
                        CHECK (operation IN ('PUSH', 'PULL', 'CONFLICT_RESOLVED')),
    entity_type     VARCHAR(30) NOT NULL
                        CHECK (entity_type IN ('financial_account', 'transaction', 'investment_asset')),
    entity_id       UUID        NOT NULL,
    client_checksum VARCHAR(64) NULL,
    server_checksum VARCHAR(64) NULL,
    resolution      VARCHAR(20) NULL
                        CHECK (resolution IN ('SERVER_WINS', 'NO_CONFLICT', NULL)),
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sync_log_entity_synced_at_idx
    ON {SCHEMA}.sync_log (entity_type, entity_id, synced_at DESC);

CREATE INDEX IF NOT EXISTS sync_log_synced_at_idx
    ON {SCHEMA}.sync_log (synced_at DESC);

COMMENT ON TABLE {SCHEMA}.sync_log IS
    'Append-only audit log for offline sync operations. Used for debugging sync conflicts.';
```

---

## 5. DDL: kasku_price {#ddl-kasku_price}

Database milik `price-service`.

### 5.1 Tabel `public.price_cache`

```sql
CREATE TABLE IF NOT EXISTS public.price_cache (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol      VARCHAR(20)     NOT NULL,
    source      VARCHAR(30)     NOT NULL CHECK (source IN ('COINGECKO', 'METALS_LIVE', 'MANUAL')),
    price_idr   NUMERIC(30, 8)  NOT NULL CHECK (price_idr > 0),
    price_usd   NUMERIC(30, 8)  NOT NULL CHECK (price_usd > 0),
    fetched_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ     NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS price_cache_symbol_source_unique_idx
    ON public.price_cache (symbol, source);

CREATE INDEX IF NOT EXISTS price_cache_expires_at_idx
    ON public.price_cache (expires_at);

COMMENT ON TABLE public.price_cache IS
    'TTL-based price cache. Default TTL: 900 seconds (15 minutes).';
COMMENT ON COLUMN public.price_cache.symbol IS
    'Asset symbol. Examples: BTC, ETH, XAU (gold), XAG (silver).';
```

**UPSERT Pattern (digunakan oleh price-service):**

```sql
INSERT INTO public.price_cache (symbol, source, price_idr, price_usd, fetched_at, expires_at)
VALUES ($1, $2, $3, $4, now(), now() + make_interval(secs => $5))
ON CONFLICT (symbol, source)
DO UPDATE SET
    price_idr  = EXCLUDED.price_idr,
    price_usd  = EXCLUDED.price_usd,
    fetched_at = EXCLUDED.fetched_at,
    expires_at = EXCLUDED.expires_at;
```

---

## 6. DDL: kasku_admin {#ddl-kasku_admin}

Database milik `admin-service`.

### 6.1 Tabel `public.admin_users`

```sql
CREATE TABLE IF NOT EXISTS public.admin_users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(30) NOT NULL,
    password_hash   TEXT        NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'SUPPORT'
                        CHECK (role IN ('SUPER_ADMIN', 'SUPPORT')),
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_users_username_unique_idx
    ON public.admin_users (LOWER(username));

CREATE INDEX IF NOT EXISTS admin_users_active_role_idx
    ON public.admin_users (is_active, role)
    WHERE is_active = true;

CREATE TRIGGER admin_users_set_updated_at
    BEFORE UPDATE ON public.admin_users
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();

COMMENT ON TABLE public.admin_users IS
    'Internal admin accounts. Separate from user accounts in kasku_auth.';
COMMENT ON COLUMN public.admin_users.password_hash IS
    'Argon2id hash. Same parameters as kasku_auth.users.password_hash.';
```

---

## 7. Migration Files Structure {#migration-files-structure}

Semua service menggunakan `golang-migrate` (Go services) atau `sqlx-migrate` (Rust services).
File migration menggunakan format `{version}_{description}.{direction}.sql`.

### 7.1 auth-service/migrations/

```
auth-service/migrations/
├── 000001_create_updated_at_function.up.sql
├── 000001_create_updated_at_function.down.sql
├── 000002_create_users.up.sql
├── 000002_create_users.down.sql
├── 000003_create_refresh_tokens.up.sql
├── 000003_create_refresh_tokens.down.sql
├── 000004_create_email_verifications.up.sql
├── 000004_create_email_verifications.down.sql
└── 000005_create_password_reset_tokens.up.sql
    000005_create_password_reset_tokens.down.sql
```

**000001_create_updated_at_function.up.sql:**
```sql
CREATE OR REPLACE FUNCTION public.set_updated_at_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

**000001_create_updated_at_function.down.sql:**
```sql
DROP FUNCTION IF EXISTS public.set_updated_at_timestamp() CASCADE;
```

**000002_create_users.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS public.users (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    email               VARCHAR(254)    NOT NULL,
    username            VARCHAR(30)     NOT NULL,
    password_hash       TEXT            NOT NULL,
    is_active           BOOLEAN         NOT NULL DEFAULT false,
    email_verified      BOOLEAN         NOT NULL DEFAULT false,
    failed_login_count  SMALLINT        NOT NULL DEFAULT 0
                            CHECK (failed_login_count >= 0 AND failed_login_count <= 10),
    locked_until        TIMESTAMPTZ     NULL,
    last_login_at       TIMESTAMPTZ     NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
    ON public.users (LOWER(email));

CREATE UNIQUE INDEX IF NOT EXISTS users_username_unique_idx
    ON public.users (LOWER(username));

CREATE INDEX IF NOT EXISTS users_is_active_idx
    ON public.users (is_active)
    WHERE is_active = true;

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON public.users
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
```

**000002_create_users.down.sql:**
```sql
DROP TRIGGER IF EXISTS users_set_updated_at ON public.users;
DROP TABLE IF EXISTS public.users CASCADE;
```

**000003_create_refresh_tokens.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS public.refresh_tokens (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID            NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)        NOT NULL,
    user_agent  VARCHAR(512)    NULL,
    ip_address  INET            NULL,
    expires_at  TIMESTAMPTZ     NOT NULL,
    is_revoked  BOOLEAN         NOT NULL DEFAULT false,
    revoked_at  TIMESTAMPTZ     NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS refresh_tokens_token_hash_unique_idx
    ON public.refresh_tokens (token_hash);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_active_idx
    ON public.refresh_tokens (user_id)
    WHERE is_revoked = false;

CREATE INDEX IF NOT EXISTS refresh_tokens_expires_at_idx
    ON public.refresh_tokens (expires_at)
    WHERE is_revoked = false;
```

**000003_create_refresh_tokens.down.sql:**
```sql
DROP TABLE IF EXISTS public.refresh_tokens CASCADE;
```

**000004_create_email_verifications.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS public.email_verifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)    NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS email_verifications_token_hash_idx
    ON public.email_verifications (token_hash);

CREATE INDEX IF NOT EXISTS email_verifications_user_id_unverified_idx
    ON public.email_verifications (user_id)
    WHERE verified_at IS NULL;
```

**000004_create_email_verifications.down.sql:**
```sql
DROP TABLE IF EXISTS public.email_verifications CASCADE;
```

**000005_create_password_reset_tokens.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS public.password_reset_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)    NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS password_reset_tokens_token_hash_idx
    ON public.password_reset_tokens (token_hash);

CREATE INDEX IF NOT EXISTS password_reset_tokens_user_id_unused_idx
    ON public.password_reset_tokens (user_id)
    WHERE used_at IS NULL;
```

**000005_create_password_reset_tokens.down.sql:**
```sql
DROP TABLE IF EXISTS public.password_reset_tokens CASCADE;
```

### 7.2 billing-service/migrations/

```
billing-service/migrations/
├── 000001_create_updated_at_function.up.sql
├── 000001_create_updated_at_function.down.sql
├── 000002_create_subscription_plans.up.sql
├── 000002_create_subscription_plans.down.sql
├── 000003_create_subscriptions.up.sql
├── 000003_create_subscriptions.down.sql
├── 000004_create_payments.up.sql
├── 000004_create_payments.down.sql
└── 000005_seed_subscription_plans.up.sql
    000005_seed_subscription_plans.down.sql
```

**000001_create_updated_at_function.up.sql:** (sama dengan auth-service)

**000002_create_subscription_plans.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS public.subscription_plans (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(20) NOT NULL CHECK (name IN ('FREE', 'BASIC', 'PRO')),
    display_name    VARCHAR(50) NOT NULL,
    price_idr       INTEGER     NOT NULL CHECK (price_idr >= 0),
    billing_cycle   VARCHAR(10) NOT NULL DEFAULT 'MONTHLY'
                        CHECK (billing_cycle IN ('MONTHLY', 'YEARLY', 'NONE')),
    limits_json     JSONB       NOT NULL,
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS subscription_plans_name_unique_idx
    ON public.subscription_plans (name);

CREATE INDEX IF NOT EXISTS subscription_plans_active_idx
    ON public.subscription_plans (is_active)
    WHERE is_active = true;

CREATE TRIGGER subscription_plans_set_updated_at
    BEFORE UPDATE ON public.subscription_plans
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
```

**000002_create_subscription_plans.down.sql:**
```sql
DROP TRIGGER IF EXISTS subscription_plans_set_updated_at ON public.subscription_plans;
DROP TABLE IF EXISTS public.subscription_plans CASCADE;
```

**000003_create_subscriptions.up.sql:**
```sql
CREATE TYPE subscription_status AS ENUM (
    'TRIALING', 'ACTIVE', 'PAST_DUE', 'CANCELLED', 'EXPIRED'
);

CREATE TABLE IF NOT EXISTS public.subscriptions (
    id                      UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID                NOT NULL,
    plan_id                 UUID                NOT NULL REFERENCES public.subscription_plans(id),
    status                  subscription_status NOT NULL DEFAULT 'ACTIVE',
    current_period_start    TIMESTAMPTZ         NOT NULL DEFAULT now(),
    current_period_end      TIMESTAMPTZ         NULL,
    cancelled_at            TIMESTAMPTZ         NULL,
    cancel_at_period_end    BOOLEAN             NOT NULL DEFAULT false,
    created_at              TIMESTAMPTZ         NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ         NOT NULL DEFAULT now(),

    CONSTRAINT subscriptions_cancel_consistency_check
        CHECK (
            (status = 'CANCELLED' AND cancelled_at IS NOT NULL)
            OR (status != 'CANCELLED' AND cancelled_at IS NULL)
        )
);

CREATE UNIQUE INDEX IF NOT EXISTS subscriptions_user_active_unique_idx
    ON public.subscriptions (user_id)
    WHERE status IN ('ACTIVE', 'TRIALING', 'PAST_DUE');

CREATE INDEX IF NOT EXISTS subscriptions_user_id_idx
    ON public.subscriptions (user_id);

CREATE INDEX IF NOT EXISTS subscriptions_status_period_end_idx
    ON public.subscriptions (status, current_period_end)
    WHERE status IN ('ACTIVE', 'TRIALING') AND current_period_end IS NOT NULL;

CREATE TRIGGER subscriptions_set_updated_at
    BEFORE UPDATE ON public.subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
```

**000003_create_subscriptions.down.sql:**
```sql
DROP TRIGGER IF EXISTS subscriptions_set_updated_at ON public.subscriptions;
DROP TABLE IF EXISTS public.subscriptions CASCADE;
DROP TYPE IF EXISTS subscription_status;
```

**000004_create_payments.up.sql:**
```sql
CREATE TYPE payment_status AS ENUM (
    'PENDING', 'SETTLEMENT', 'EXPIRE', 'CANCEL', 'DENY', 'REFUND', 'FAILURE'
);

CREATE TABLE IF NOT EXISTS public.payments (
    id                          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                     UUID            NOT NULL,
    subscription_id             UUID            NULL REFERENCES public.subscriptions(id)
                                    ON DELETE SET NULL,
    midtrans_order_id           VARCHAR(100)    NOT NULL,
    midtrans_transaction_id     VARCHAR(100)    NULL,
    amount_idr                  INTEGER         NOT NULL CHECK (amount_idr > 0),
    status                      payment_status  NOT NULL DEFAULT 'PENDING',
    payment_method              VARCHAR(50)     NULL,
    snap_token                  VARCHAR(255)    NULL,
    paid_at                     TIMESTAMPTZ     NULL,
    expired_at                  TIMESTAMPTZ     NULL,
    created_at                  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ     NOT NULL DEFAULT now(),

    CONSTRAINT payments_order_id_format_check
        CHECK (midtrans_order_id ~ '^KASKU-[0-9a-f]{8}-[0-9]+$'),
    CONSTRAINT payments_settlement_requires_paid_at_check
        CHECK (
            (status = 'SETTLEMENT' AND paid_at IS NOT NULL)
            OR (status != 'SETTLEMENT')
        )
);

CREATE UNIQUE INDEX IF NOT EXISTS payments_midtrans_order_id_unique_idx
    ON public.payments (midtrans_order_id);

CREATE INDEX IF NOT EXISTS payments_user_id_status_idx
    ON public.payments (user_id, status);

CREATE INDEX IF NOT EXISTS payments_midtrans_transaction_id_idx
    ON public.payments (midtrans_transaction_id)
    WHERE midtrans_transaction_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS payments_created_at_idx
    ON public.payments (created_at DESC);

CREATE TRIGGER payments_set_updated_at
    BEFORE UPDATE ON public.payments
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
```

**000004_create_payments.down.sql:**
```sql
DROP TRIGGER IF EXISTS payments_set_updated_at ON public.payments;
DROP TABLE IF EXISTS public.payments CASCADE;
DROP TYPE IF EXISTS payment_status;
```

**000005_seed_subscription_plans.up.sql:** (isi seed dari section 3.2 di atas)

**000005_seed_subscription_plans.down.sql:**
```sql
DELETE FROM public.subscription_plans
WHERE id IN (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000003'
);
```

### 7.3 finance-service/migrations/

Untuk `kasku_finance`, migration tidak di-run via `golang-migrate` karena schema per tenant.
Sebagai gantinya, DDL template disimpan sebagai migration file yang digunakan oleh
`provision_tenant` PostgreSQL function.

```
finance-service/migrations/
└── 000001_tenant_schema_template.up.sql
    000001_tenant_schema_template.down.sql
```

**000001_tenant_schema_template.up.sql:**
File ini berisi PostgreSQL function `provision_tenant` (lihat Section 8).
Dijalankan SEKALI saat service pertama kali startup untuk mendaftarkan function di DB.

**000001_tenant_schema_template.down.sql:**
```sql
DROP FUNCTION IF EXISTS provision_tenant(UUID);
DROP FUNCTION IF EXISTS deprovision_tenant(UUID);
```

### 7.4 price-service/migrations/ dan admin-service/migrations/

```
price-service/migrations/
├── 000001_create_price_cache.up.sql    # DDL dari Section 5.1
└── 000001_create_price_cache.down.sql  # DROP TABLE public.price_cache

admin-service/migrations/
├── 000001_create_updated_at_function.up.sql
├── 000001_create_updated_at_function.down.sql
├── 000002_create_admin_users.up.sql    # DDL dari Section 6.1
└── 000002_create_admin_users.down.sql  # DROP TABLE public.admin_users
```

---

## 8. Tenant Schema Provisioning Function {#tenant-schema-provisioning-function}

Function PostgreSQL ini dipanggil oleh `user-service` saat user baru register.
Dijalankan di database `kasku_finance`.

```sql
CREATE OR REPLACE FUNCTION provision_tenant(p_user_id UUID)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_schema_name TEXT;
    v_sanitized_uuid TEXT;
BEGIN
    -- Sanitize UUID: ganti '-' dengan '_' untuk nama schema yang valid
    v_sanitized_uuid := REPLACE(p_user_id::text, '-', '_');
    v_schema_name := 'tenant_' || v_sanitized_uuid;

    -- Validasi: pastikan user_id valid UUID format
    IF p_user_id IS NULL THEN
        RAISE EXCEPTION 'provision_tenant: p_user_id cannot be null';
    END IF;

    -- Buat schema (idempotent)
    EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', v_schema_name);

    -- Pastikan set_updated_at_timestamp function ada di public schema
    -- (function ini dibuat oleh finance-service migration 000001)
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc p
        JOIN pg_namespace n ON p.pronamespace = n.oid
        WHERE n.nspname = 'public' AND p.proname = 'set_updated_at_timestamp'
    ) THEN
        RAISE EXCEPTION 'provision_tenant: public.set_updated_at_timestamp function not found. Run migrations first.';
    END IF;

    -- DDL: financial_accounts
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.financial_accounts (
            id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
            name            VARCHAR(100)    NOT NULL,
            account_type    VARCHAR(20)     NOT NULL
                                CHECK (account_type IN (''BANK'', ''EWALLET'', ''CASH'', ''INVESTMENT'')),
            currency        CHAR(3)         NOT NULL DEFAULT ''IDR'',
            balance         NUMERIC(20, 2)  NOT NULL DEFAULT 0.00,
            institution     VARCHAR(100)    NULL,
            account_number  VARCHAR(50)     NULL,
            color           CHAR(7)         NULL CHECK (color ~ ''^#[0-9A-Fa-f]{6}$''),
            icon            VARCHAR(50)     NULL,
            is_active       BOOLEAN         NOT NULL DEFAULT true,
            is_deleted      BOOLEAN         NOT NULL DEFAULT false,
            deleted_at      TIMESTAMPTZ     NULL,
            sort_order      SMALLINT        NOT NULL DEFAULT 0,
            created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
            updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
            CONSTRAINT financial_accounts_delete_consistency_check
                CHECK (
                    (is_deleted = true AND deleted_at IS NOT NULL)
                    OR (is_deleted = false AND deleted_at IS NULL)
                )
        )', v_schema_name);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS financial_accounts_active_idx
            ON %I.financial_accounts (is_deleted, is_active)
            WHERE is_deleted = false', v_schema_name);

    EXECUTE format('
        CREATE TRIGGER financial_accounts_set_updated_at
            BEFORE UPDATE ON %I.financial_accounts
            FOR EACH ROW
            EXECUTE FUNCTION public.set_updated_at_timestamp()', v_schema_name);

    -- DDL: balance_history
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.balance_history (
            id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
            account_id      UUID            NOT NULL REFERENCES %I.financial_accounts(id)
                                ON DELETE CASCADE,
            balance_before  NUMERIC(20, 2)  NOT NULL,
            balance_after   NUMERIC(20, 2)  NOT NULL,
            change_amount   NUMERIC(20, 2)  NOT NULL
                                GENERATED ALWAYS AS (balance_after - balance_before) STORED,
            change_reason   VARCHAR(50)     NOT NULL
                                CHECK (change_reason IN (''TRANSACTION'', ''MANUAL_ADJUST'', ''SYNC'', ''INITIAL'')),
            reference_id    UUID            NULL,
            recorded_at     TIMESTAMPTZ     NOT NULL DEFAULT now()
        )', v_schema_name, v_schema_name);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS balance_history_account_id_recorded_at_idx
            ON %I.balance_history (account_id, recorded_at DESC)', v_schema_name);

    -- DDL: categories
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.categories (
            id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            name        VARCHAR(50) NOT NULL,
            type        VARCHAR(10) NOT NULL CHECK (type IN (''INCOME'', ''EXPENSE'', ''TRANSFER'')),
            icon        VARCHAR(50) NULL,
            color       CHAR(7)     NULL CHECK (color ~ ''^#[0-9A-Fa-f]{6}$''),
            is_default  BOOLEAN     NOT NULL DEFAULT false,
            is_deleted  BOOLEAN     NOT NULL DEFAULT false,
            deleted_at  TIMESTAMPTZ NULL,
            sort_order  SMALLINT    NOT NULL DEFAULT 0,
            created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
        )', v_schema_name);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS categories_type_active_idx
            ON %I.categories (type, is_deleted)
            WHERE is_deleted = false', v_schema_name);

    EXECUTE format('
        CREATE TRIGGER categories_set_updated_at
            BEFORE UPDATE ON %I.categories
            FOR EACH ROW
            EXECUTE FUNCTION public.set_updated_at_timestamp()', v_schema_name);

    -- Seed categories default
    EXECUTE format('
        INSERT INTO %I.categories (name, type, icon, color, is_default, sort_order)
        VALUES
            (''Makanan & Minuman'',   ''EXPENSE'', ''utensils'',       ''#F97316'', true, 1),
            (''Transport'',           ''EXPENSE'', ''car'',            ''#3B82F6'', true, 2),
            (''Belanja'',             ''EXPENSE'', ''shopping-bag'',   ''#8B5CF6'', true, 3),
            (''Tagihan & Utilitas'',  ''EXPENSE'', ''zap'',            ''#EF4444'', true, 4),
            (''Kesehatan'',           ''EXPENSE'', ''heart'',          ''#EC4899'', true, 5),
            (''Hiburan'',             ''EXPENSE'', ''music'',          ''#F59E0B'', true, 6),
            (''Pendidikan'',          ''EXPENSE'', ''book'',           ''#06B6D4'', true, 7),
            (''Lain-lain'',           ''EXPENSE'', ''more-horizontal'',''#6B7280'', true, 8),
            (''Gaji'',                ''INCOME'',  ''briefcase'',      ''#10B981'', true, 9),
            (''Freelance'',           ''INCOME'',  ''laptop'',         ''#34D399'', true, 10),
            (''Investasi'',           ''INCOME'',  ''trending-up'',    ''#6EE7B7'', true, 11),
            (''Lain-lain'',           ''INCOME'',  ''plus-circle'',    ''#A7F3D0'', true, 12),
            (''Transfer Masuk'',      ''TRANSFER'',''arrow-down-left'',''#60A5FA'', true, 13),
            (''Transfer Keluar'',     ''TRANSFER'',''arrow-up-right'', ''#93C5FD'', true, 14)
        ON CONFLICT DO NOTHING', v_schema_name);

    -- DDL: transactions
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.transactions (
            id               UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
            sync_id          UUID            NULL,
            account_id       UUID            NOT NULL REFERENCES %I.financial_accounts(id)
                                 ON DELETE RESTRICT,
            category_id      UUID            NOT NULL REFERENCES %I.categories(id)
                                 ON DELETE RESTRICT,
            transaction_type VARCHAR(10)     NOT NULL
                                 CHECK (transaction_type IN (''INCOME'', ''EXPENSE'', ''TRANSFER'')),
            amount           NUMERIC(20, 2)  NOT NULL CHECK (amount > 0),
            description      VARCHAR(500)    NULL,
            notes            TEXT            NULL,
            transaction_date DATE            NOT NULL,
            to_account_id    UUID            NULL REFERENCES %I.financial_accounts(id)
                                 ON DELETE RESTRICT,
            is_deleted       BOOLEAN         NOT NULL DEFAULT false,
            deleted_at       TIMESTAMPTZ     NULL,
            created_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
            updated_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
            CONSTRAINT transactions_transfer_requires_to_account_check
                CHECK (
                    (transaction_type = ''TRANSFER'' AND to_account_id IS NOT NULL)
                    OR (transaction_type != ''TRANSFER'' AND to_account_id IS NULL)
                ),
            CONSTRAINT transactions_delete_consistency_check
                CHECK (
                    (is_deleted = true AND deleted_at IS NOT NULL)
                    OR (is_deleted = false AND deleted_at IS NULL)
                )
        )', v_schema_name, v_schema_name, v_schema_name, v_schema_name);

    EXECUTE format('
        CREATE UNIQUE INDEX IF NOT EXISTS transactions_sync_id_unique_idx
            ON %I.transactions (sync_id)
            WHERE sync_id IS NOT NULL AND is_deleted = false', v_schema_name);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS transactions_account_date_idx
            ON %I.transactions (account_id, transaction_date DESC)
            WHERE is_deleted = false', v_schema_name);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS transactions_date_type_idx
            ON %I.transactions (transaction_date DESC, transaction_type)
            WHERE is_deleted = false', v_schema_name);

    EXECUTE format('
        CREATE TRIGGER transactions_set_updated_at
            BEFORE UPDATE ON %I.transactions
            FOR EACH ROW
            EXECUTE FUNCTION public.set_updated_at_timestamp()', v_schema_name);

    -- DDL: investment_assets
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.investment_assets (
            id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
            name            VARCHAR(100)    NOT NULL,
            asset_type      VARCHAR(20)     NOT NULL
                                CHECK (asset_type IN (''CRYPTO'', ''GOLD'', ''STOCK'', ''MUTUAL_FUND'', ''OTHER'')),
            symbol          VARCHAR(20)     NOT NULL,
            quantity        NUMERIC(30, 10) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
            avg_buy_price   NUMERIC(20, 2)  NOT NULL DEFAULT 0 CHECK (avg_buy_price >= 0),
            currency        CHAR(3)         NOT NULL DEFAULT ''IDR'',
            is_deleted      BOOLEAN         NOT NULL DEFAULT false,
            deleted_at      TIMESTAMPTZ     NULL,
            sort_order      SMALLINT        NOT NULL DEFAULT 0,
            created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
            updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
        )', v_schema_name);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS investment_assets_active_idx
            ON %I.investment_assets (is_deleted)
            WHERE is_deleted = false', v_schema_name);

    EXECUTE format('
        CREATE TRIGGER investment_assets_set_updated_at
            BEFORE UPDATE ON %I.investment_assets
            FOR EACH ROW
            EXECUTE FUNCTION public.set_updated_at_timestamp()', v_schema_name);

    -- DDL: unit_history
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.unit_history (
            id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
            asset_id            UUID            NOT NULL REFERENCES %I.investment_assets(id)
                                    ON DELETE CASCADE,
            transaction_type    VARCHAR(10)     NOT NULL CHECK (transaction_type IN (''BUY'', ''SELL'', ''ADJUST'')),
            quantity_change     NUMERIC(30, 10) NOT NULL,
            price_per_unit      NUMERIC(20, 2)  NOT NULL CHECK (price_per_unit > 0),
            total_value         NUMERIC(20, 2)  NOT NULL
                                    GENERATED ALWAYS AS (ABS(quantity_change) * price_per_unit) STORED,
            notes               VARCHAR(500)    NULL,
            transaction_date    DATE            NOT NULL,
            recorded_at         TIMESTAMPTZ     NOT NULL DEFAULT now()
        )', v_schema_name, v_schema_name);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS unit_history_asset_id_date_idx
            ON %I.unit_history (asset_id, transaction_date DESC)', v_schema_name);

    -- DDL: sync_log
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.sync_log (
            id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            operation       VARCHAR(20) NOT NULL
                                CHECK (operation IN (''PUSH'', ''PULL'', ''CONFLICT_RESOLVED'')),
            entity_type     VARCHAR(30) NOT NULL
                                CHECK (entity_type IN (''financial_account'', ''transaction'', ''investment_asset'')),
            entity_id       UUID        NOT NULL,
            client_checksum VARCHAR(64) NULL,
            server_checksum VARCHAR(64) NULL,
            resolution      VARCHAR(20) NULL
                                CHECK (resolution IN (''SERVER_WINS'', ''NO_CONFLICT'', NULL)),
            synced_at       TIMESTAMPTZ NOT NULL DEFAULT now()
        )', v_schema_name);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS sync_log_entity_synced_at_idx
            ON %I.sync_log (entity_type, entity_id, synced_at DESC)', v_schema_name);

    RAISE NOTICE 'provision_tenant: schema % created successfully for user %', v_schema_name, p_user_id;
END;
$$;

COMMENT ON FUNCTION provision_tenant(UUID) IS
    'Creates a new tenant schema with all required tables and seed data. Idempotent: safe to call multiple times.';
```

### Deprovision Function (untuk GDPR Right to Erasure)

```sql
CREATE OR REPLACE FUNCTION deprovision_tenant(p_user_id UUID)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_schema_name TEXT;
BEGIN
    v_schema_name := 'tenant_' || REPLACE(p_user_id::text, '-', '_');

    -- Drop schema dan semua objek di dalamnya
    EXECUTE format('DROP SCHEMA IF EXISTS %I CASCADE', v_schema_name);

    RAISE NOTICE 'deprovision_tenant: schema % dropped for user %', v_schema_name, p_user_id;
END;
$$;

COMMENT ON FUNCTION deprovision_tenant(UUID) IS
    'Drops tenant schema and all data. Used for GDPR right to erasure. IRREVERSIBLE.';
```

---

## 9. Index Strategy {#index-strategy}

### kasku_auth

| Index | Tabel | Kolom | Tipe | Query Pattern |
|-------|-------|-------|------|---------------|
| `users_email_unique_idx` | users | `LOWER(email)` | Unique | Login by email, register duplicate check |
| `users_username_unique_idx` | users | `LOWER(username)` | Unique | Register duplicate check |
| `users_is_active_idx` | users | `is_active` WHERE true | Partial | Filter active users (admin queries) |
| `refresh_tokens_token_hash_unique_idx` | refresh_tokens | `token_hash` | Unique | Token lookup on refresh |
| `refresh_tokens_user_id_active_idx` | refresh_tokens | `user_id` WHERE not revoked | Partial | Get active tokens per user, revoke all |
| `refresh_tokens_expires_at_idx` | refresh_tokens | `expires_at` WHERE not revoked | Partial | Cleanup job: delete expired tokens |

### kasku_billing

| Index | Tabel | Kolom | Tipe | Query Pattern |
|-------|-------|-------|------|---------------|
| `subscription_plans_name_unique_idx` | subscription_plans | `name` | Unique | Lookup plan by name |
| `subscriptions_user_active_unique_idx` | subscriptions | `user_id` WHERE active | Partial Unique | Ensure max 1 active subscription per user |
| `subscriptions_user_id_idx` | subscriptions | `user_id` | B-tree | Get all subscriptions for a user |
| `subscriptions_status_period_end_idx` | subscriptions | `(status, current_period_end)` | Composite Partial | Cron job: find expiring subscriptions |
| `payments_midtrans_order_id_unique_idx` | payments | `midtrans_order_id` | Unique | Webhook idempotency check |
| `payments_user_id_status_idx` | payments | `(user_id, status)` | Composite | Invoice list per user, filter by status |
| `payments_midtrans_transaction_id_idx` | payments | `midtrans_transaction_id` | Partial | Lookup by Midtrans transaction ID |
| `payments_created_at_idx` | payments | `created_at DESC` | B-tree | Admin: recent payments list |

### kasku_finance (per tenant schema)

| Index | Tabel | Kolom | Tipe | Query Pattern |
|-------|-------|-------|------|---------------|
| `financial_accounts_active_idx` | financial_accounts | `(is_deleted, is_active)` WHERE not deleted | Partial | List active accounts |
| `categories_type_active_idx` | categories | `(type, is_deleted)` WHERE not deleted | Partial | Get categories by type |
| `transactions_sync_id_unique_idx` | transactions | `sync_id` WHERE not null and not deleted | Partial Unique | Sync idempotency check |
| `transactions_account_date_idx` | transactions | `(account_id, transaction_date DESC)` WHERE not deleted | Composite Partial | Transactions per account, chronological |
| `transactions_date_type_idx` | transactions | `(transaction_date DESC, transaction_type)` WHERE not deleted | Composite Partial | Dashboard: transactions this month by type |
| `investment_assets_active_idx` | investment_assets | `is_deleted` WHERE false | Partial | List active investment assets |
| `balance_history_account_id_recorded_at_idx` | balance_history | `(account_id, recorded_at DESC)` | Composite | Balance history per account |
| `unit_history_asset_id_date_idx` | unit_history | `(asset_id, transaction_date DESC)` | Composite | Unit history per asset |
| `sync_log_entity_synced_at_idx` | sync_log | `(entity_type, entity_id, synced_at DESC)` | Composite | Audit: sync history per entity |

### kasku_price

| Index | Tabel | Kolom | Tipe | Query Pattern |
|-------|-------|-------|------|---------------|
| `price_cache_symbol_source_unique_idx` | price_cache | `(symbol, source)` | Unique | UPSERT cache, lookup by symbol+source |
| `price_cache_expires_at_idx` | price_cache | `expires_at` | B-tree | Cleanup job: delete expired cache |

---

## 10. Database Security — Least Privilege {#database-security}

Setiap service memiliki dedicated PostgreSQL user dengan privilege minimal.

```sql
-- ============================================================
-- kasku_auth database users
-- ============================================================

CREATE USER kasku_auth_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_auth TO kasku_auth_svc;
GRANT USAGE ON SCHEMA public TO kasku_auth_svc;
GRANT SELECT, INSERT, UPDATE, DELETE
    ON TABLE public.users,
             public.refresh_tokens,
             public.email_verifications,
             public.password_reset_tokens
    TO kasku_auth_svc;
-- Tidak ada DROP, ALTER, TRUNCATE, CREATE untuk auth service

-- ============================================================
-- kasku_billing database users
-- ============================================================

CREATE USER kasku_billing_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_billing TO kasku_billing_svc;
GRANT USAGE ON SCHEMA public TO kasku_billing_svc;
GRANT SELECT, INSERT, UPDATE, DELETE
    ON TABLE public.subscription_plans,
             public.subscriptions,
             public.payments
    TO kasku_billing_svc;
-- billing-service boleh UPDATE subscription_plans (admin update limits),
-- tapi tidak DROP TABLE

-- ============================================================
-- kasku_finance database users (service-specific)
-- ============================================================

-- finance-service: hanya akses financial_accounts dan balance_history
CREATE USER kasku_finance_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_finance TO kasku_finance_svc;
GRANT USAGE ON SCHEMA public TO kasku_finance_svc;
-- finance-service tidak tahu schema name saat connect, perlu set search_path di query
-- Untuk dynamic schema access: grant di-apply via function atau via SET search_path per query

-- transaction-service: hanya akses transactions dan categories
CREATE USER kasku_transaction_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_finance TO kasku_transaction_svc;
GRANT USAGE ON SCHEMA public TO kasku_transaction_svc;

-- investment-service: hanya akses investment_assets dan unit_history
CREATE USER kasku_investment_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_finance TO kasku_investment_svc;
GRANT USAGE ON SCHEMA public TO kasku_investment_svc;

-- sync-service: akses semua tabel di tenant schema (read + write untuk sync)
CREATE USER kasku_sync_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_finance TO kasku_sync_svc;
GRANT USAGE ON SCHEMA public TO kasku_sync_svc;

-- user-service: hanya execute privilege untuk provision_tenant function
CREATE USER kasku_user_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_finance TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION provision_tenant(UUID) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION deprovision_tenant(UUID) TO kasku_user_svc;
-- user-service TIDAK punya GRANT ke tabel apapun di kasku_finance
-- (akses dilakukan via SECURITY DEFINER function provision_tenant)

-- ============================================================
-- Dynamic schema GRANT untuk service yang akses tenant schema
-- ============================================================
-- Problem: grant per-tabel tidak feasible karena nama schema dinamis.
-- Solusi: gunakan event trigger untuk auto-grant saat schema baru dibuat,
--         ATAU gunakan SECURITY DEFINER pada provision_tenant yang sudah
--         set search_path, sehingga service user tidak perlu akses langsung.
--
-- Pendekatan yang dipilih: application-level schema prefix di setiap query.
-- Service user (kasku_finance_svc, etc) mendapat GRANT via PostgreSQL superuser
-- pada setiap schema baru via event trigger:

CREATE OR REPLACE FUNCTION grant_tenant_schema_access()
RETURNS event_trigger
LANGUAGE plpgsql
AS $$
DECLARE
    obj record;
    v_schema_name TEXT;
BEGIN
    FOR obj IN SELECT * FROM pg_event_trigger_ddl_commands()
        WHERE command_tag = 'CREATE SCHEMA'
    LOOP
        v_schema_name := obj.object_identity;
        IF v_schema_name LIKE 'tenant_%' THEN
            EXECUTE format('GRANT USAGE ON SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema_name);
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_finance_svc', v_schema_name);
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_transaction_svc', v_schema_name);
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_investment_svc', v_schema_name);
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_sync_svc', v_schema_name);
        END IF;
    END LOOP;
END;
$$;

CREATE EVENT TRIGGER grant_new_tenant_schema_access
    ON ddl_command_end
    WHEN TAG IN ('CREATE SCHEMA')
    EXECUTE FUNCTION grant_tenant_schema_access();

-- ============================================================
-- kasku_price database users
-- ============================================================

CREATE USER kasku_price_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_price TO kasku_price_svc;
GRANT USAGE ON SCHEMA public TO kasku_price_svc;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE public.price_cache TO kasku_price_svc;

-- ============================================================
-- kasku_admin database users
-- ============================================================

CREATE USER kasku_admin_svc WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_admin TO kasku_admin_svc;
GRANT USAGE ON SCHEMA public TO kasku_admin_svc;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE public.admin_users TO kasku_admin_svc;

-- admin-service juga butuh READ ONLY ke kasku_auth dan kasku_billing
CREATE USER kasku_admin_read WITH PASSWORD 'CHANGE_IN_PRODUCTION_VIA_ENV';
GRANT CONNECT ON DATABASE kasku_auth TO kasku_admin_read;
GRANT CONNECT ON DATABASE kasku_billing TO kasku_admin_read;
GRANT USAGE ON SCHEMA public TO kasku_admin_read;  -- run on both databases
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kasku_admin_read;  -- run on both databases
```

---

## 11. Tier Limit Enforcement Pattern {#tier-limit-enforcement-pattern}

### 11.1 Format `limits_json`

Kolom `subscription_plans.limits_json` menggunakan schema berikut:

```json
{
  "max_transactions_per_month": 50,
  "max_financial_accounts": 3,
  "max_investment_instruments": 2,
  "history_retention_months": 3,
  "email_notifications_enabled": false,
  "export_csv_enabled": false,
  "offline_sync_enabled": true,
  "realtime_price_enabled": true
}
```

Nilai `-1` berarti unlimited (hanya di PRO tier).

### 11.2 Pengambilan Limits di api-gateway

api-gateway memanggil billing-service via gRPC `GetUserTierLimits(user_id)` untuk
mendapatkan limits aktif user, kemudian meng-inject ke request header sebelum di-proxy
ke downstream service:

```
X-Tier-Max-Transactions: 50
X-Tier-Max-Accounts: 3
X-Tier-Max-Investments: 2
X-Tier-History-Months: 3
X-Tier-Email-Notifications: false
X-Tier-Export-Csv: false
```

Downstream service (finance-service, transaction-service, investment-service) membaca
header ini untuk enforcement tanpa perlu query ke billing-service sendiri.

### 11.3 Enforcement Query Pattern di Application Layer

**finance-service — cek limit sebelum CREATE account:**

```sql
-- Hitung jumlah akun aktif saat ini
SELECT COUNT(*) AS active_account_count
FROM {tenant_schema}.financial_accounts
WHERE is_deleted = false AND is_active = true;

-- Bandingkan dengan X-Tier-Max-Accounts dari header
-- Jika active_account_count >= max_accounts (dan max_accounts != -1):
-- Return HTTP 402 Payment Required dengan body TierLimitError
```

**transaction-service — cek limit sebelum CREATE transaction:**

```sql
-- Hitung transaksi di bulan berjalan
SELECT COUNT(*) AS monthly_transaction_count
FROM {tenant_schema}.transactions
WHERE is_deleted = false
  AND transaction_date >= date_trunc('month', CURRENT_DATE)
  AND transaction_date < date_trunc('month', CURRENT_DATE) + INTERVAL '1 month';

-- Bandingkan dengan X-Tier-Max-Transactions dari header
```

**investment-service — cek limit sebelum CREATE asset:**

```sql
-- Hitung instrumen investasi aktif
SELECT COUNT(*) AS active_instrument_count
FROM {tenant_schema}.investment_assets
WHERE is_deleted = false;

-- Bandingkan dengan X-Tier-Max-Investments dari header
```

### 11.4 HTTP 402 Response untuk Limit Exceeded

Semua service mengembalikan HTTP 402 Payment Required dengan format:

```json
{
  "success": false,
  "error": {
    "code": "TIER_LIMIT_EXCEEDED",
    "message": "Batas transaksi bulanan untuk plan Free telah tercapai (50/50).",
    "details": {
      "limit_type": "max_transactions_per_month",
      "current_count": 50,
      "limit": 50,
      "current_plan": "FREE",
      "upgrade_url": "https://app.kasku.id/billing/plans"
    }
  }
}
```

### 11.5 History Retention Enforcement

Untuk tier FREE (retention 3 bulan), transaction-service dan finance-service
menerapkan soft-filter pada query GET: hanya mengembalikan data dalam window retention.

```sql
-- Contoh untuk FREE tier (history_retention_months = 3)
SELECT * FROM {tenant_schema}.transactions
WHERE is_deleted = false
  AND transaction_date >= CURRENT_DATE - INTERVAL '3 months'
ORDER BY transaction_date DESC;
```

Data lama tidak dihapus dari database (data retention untuk compliance) — hanya
tidak ditampilkan kepada user berdasarkan tier aktif mereka.

---

*Dokumen ini adalah source of truth database schema KasKu SaaS v2.0.*
*Setiap perubahan schema WAJIB disertai migration file (up + down).*
*Terakhir diperbarui: 2026-04-27*
