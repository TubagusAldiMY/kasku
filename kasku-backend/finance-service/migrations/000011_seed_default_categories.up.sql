-- 000011: Seed default categories per tenant.
-- Restores default category data (dihapus di 000005) dan update provision_tenant()
-- agar tenant baru juga auto-seed saat registrasi.

-- ─── 1. Helper function ────────────────────────────────────────────────────────
-- Idempotent: skip baris yang (name, category_type, is_default=true) sudah ada.
-- Dipanggil dari provision_tenant() (tenant baru) dan dari DO block di bawah (backfill).

CREATE OR REPLACE FUNCTION public.seed_default_categories(p_tenant_schema TEXT)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    IF p_tenant_schema !~ '^tenant_[0-9a-f_]{32,36}$' THEN
        RAISE EXCEPTION 'invalid tenant schema: %', p_tenant_schema;
    END IF;

    -- UPSERT: INSERT baru atau restore + update icon/color yang sudah ada (termasuk soft-deleted).
    -- ON CONFLICT menggunakan partial unique index categories_default_unique_idx.
    EXECUTE format('
        INSERT INTO %I.categories (id, name, icon, color, category_type, is_default, is_deleted, created_at, updated_at)
        VALUES
            (gen_random_uuid(), ''Gaji'',         ''💼'', ''#10b981'', ''INCOME'',  true, false, now(), now()),
            (gen_random_uuid(), ''Bisnis'',       ''🏪'', ''#3b82f6'', ''INCOME'',  true, false, now(), now()),
            (gen_random_uuid(), ''Investasi'',    ''📈'', ''#8b5cf6'', ''INCOME'',  true, false, now(), now()),
            (gen_random_uuid(), ''Bonus'',        ''🎁'', ''#f59e0b'', ''INCOME'',  true, false, now(), now()),
            (gen_random_uuid(), ''Makanan'',      ''🍔'', ''#ef4444'', ''EXPENSE'', true, false, now(), now()),
            (gen_random_uuid(), ''Transportasi'', ''🚗'', ''#f97316'', ''EXPENSE'', true, false, now(), now()),
            (gen_random_uuid(), ''Belanja'',      ''🛒'', ''#ec4899'', ''EXPENSE'', true, false, now(), now()),
            (gen_random_uuid(), ''Kesehatan'',    ''💊'', ''#14b8a6'', ''EXPENSE'', true, false, now(), now()),
            (gen_random_uuid(), ''Tagihan'',      ''💡'', ''#64748b'', ''EXPENSE'', true, false, now(), now()),
            (gen_random_uuid(), ''Hiburan'',      ''🎮'', ''#a855f7'', ''EXPENSE'', true, false, now(), now()),
            (gen_random_uuid(), ''Pendidikan'',   ''📚'', ''#0ea5e9'', ''EXPENSE'', true, false, now(), now()),
            (gen_random_uuid(), ''Tabungan'',     ''🐷'', ''#22c55e'', ''BOTH'',    true, false, now(), now()),
            (gen_random_uuid(), ''Transfer'',     ''🔄'', ''#94a3b8'', ''BOTH'',    true, false, now(), now()),
            (gen_random_uuid(), ''Lainnya'',      ''📌'', ''#6b7280'', ''BOTH'',    true, false, now(), now())
        ON CONFLICT (name, category_type) WHERE is_default = true
        DO UPDATE SET
            icon       = EXCLUDED.icon,
            color      = EXCLUDED.color,
            is_deleted = false,
            deleted_at = NULL,
            updated_at = now()
    ', p_tenant_schema);
END;
$$;

GRANT EXECUTE ON FUNCTION public.seed_default_categories(TEXT) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION public.seed_default_categories(TEXT) TO kasku_finance_svc;

-- ─── 2. Update provision_tenant() ─────────────────────────────────────────────
-- Tambahkan pemanggilan seed_default_categories() setelah tabel categories dibuat.
-- Basis: versi 000003 (dengan investment tables) + seed call.

CREATE OR REPLACE FUNCTION public.provision_tenant(p_user_id UUID)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_schema TEXT;
BEGIN
    v_schema := 'tenant_' || replace(p_user_id::text, '-', '_');

    EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', v_schema);

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.financial_accounts (
            id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id       UUID         NOT NULL,
            name          VARCHAR(100) NOT NULL,
            account_type  VARCHAR(20)  NOT NULL DEFAULT ''BANK'',
            balance       BIGINT       NOT NULL DEFAULT 0,
            currency      CHAR(3)      NOT NULL DEFAULT ''IDR'',
            color         VARCHAR(7)   NOT NULL DEFAULT ''#6366f1'',
            icon          VARCHAR(50)  NOT NULL DEFAULT ''wallet'',
            is_default    BOOLEAN      NOT NULL DEFAULT false,
            is_deleted    BOOLEAN      NOT NULL DEFAULT false,
            deleted_at    TIMESTAMPTZ  NULL,
            created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
            updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.transactions (
            id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
            sync_id           VARCHAR(100) UNIQUE,
            account_id        UUID         NOT NULL,
            category_id       UUID         NULL,
            transaction_type  VARCHAR(20)  NOT NULL,
            amount_idr        BIGINT       NOT NULL,
            transaction_date  DATE         NOT NULL DEFAULT CURRENT_DATE,
            notes             TEXT         NULL,
            to_account_id     UUID         NULL,
            budget_id         UUID         NULL,
            is_deleted        BOOLEAN      NOT NULL DEFAULT false,
            deleted_at        TIMESTAMPTZ  NULL,
            created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
            updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.categories (
            id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
            name           VARCHAR(50)  NOT NULL,
            icon           VARCHAR(50)  NOT NULL DEFAULT ''tag'',
            color          VARCHAR(7)   NOT NULL DEFAULT ''#6366f1'',
            category_type  VARCHAR(20)  NOT NULL DEFAULT ''BOTH'',
            is_default     BOOLEAN      NOT NULL DEFAULT false,
            is_deleted     BOOLEAN      NOT NULL DEFAULT false,
            deleted_at     TIMESTAMPTZ  NULL,
            created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
            updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.balance_history (
            id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            account_id  UUID        NOT NULL,
            amount      BIGINT      NOT NULL,
            balance     BIGINT      NOT NULL,
            note        TEXT        NULL,
            created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.budgets (
            id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id         UUID        NOT NULL,
            name            VARCHAR(100) NOT NULL,
            amount          BIGINT      NOT NULL,
            daily_limit     BIGINT      NULL,
            period_type     VARCHAR(20) NOT NULL DEFAULT ''MONTHLY''
                              CHECK (period_type IN (''MONTHLY'', ''WEEKLY'', ''CUSTOM'')),
            start_date      DATE        NOT NULL,
            end_date        DATE        NULL,
            category_id     UUID        NULL,
            is_deleted      BOOLEAN     NOT NULL DEFAULT false,
            deleted_at      TIMESTAMPTZ NULL,
            created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.investment_assets (
            id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
            name            VARCHAR(100)    NOT NULL,
            asset_type      VARCHAR(20)     NOT NULL
                                              CHECK (asset_type IN (''CRYPTO'',''GOLD'',''STOCK'',''MUTUAL_FUND'',''OTHER'')),
            symbol          VARCHAR(50)     NOT NULL,
            quantity        NUMERIC(28, 8)  NOT NULL DEFAULT 0,
            avg_buy_price   NUMERIC(20, 4)  NOT NULL DEFAULT 0,
            currency        CHAR(3)         NOT NULL DEFAULT ''IDR'',
            is_deleted      BOOLEAN         NOT NULL DEFAULT false,
            deleted_at      TIMESTAMPTZ     NULL,
            sort_order      INTEGER         NOT NULL DEFAULT 0,
            created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
            updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS idx_investment_assets_active
        ON %I.investment_assets (sort_order ASC, created_at ASC)
        WHERE is_deleted = false', v_schema);

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.unit_history (
            id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
            asset_id            UUID            NOT NULL,
            transaction_type    VARCHAR(20)     NOT NULL
                                                  CHECK (transaction_type IN (''BUY'',''SELL'',''ADJUST'')),
            quantity_change     NUMERIC(28, 8)  NOT NULL,
            price_per_unit      NUMERIC(20, 4)  NOT NULL,
            notes               TEXT            NULL,
            transaction_date    DATE            NOT NULL,
            recorded_at         TIMESTAMPTZ     NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS idx_unit_history_asset_recorded
        ON %I.unit_history (asset_id, recorded_at DESC)', v_schema);

    EXECUTE format('GRANT USAGE ON SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);

    -- Seed default categories untuk tenant baru
    PERFORM public.seed_default_categories(v_schema);
END;
$$;

GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_finance_svc;

-- ─── 3. Backfill semua tenant yang sudah ada ──────────────────────────────────

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^tenant_[0-9a-f_]{32,36}$'
    LOOP
        PERFORM public.seed_default_categories(r.schema_name);
    END LOOP;
END $$;
