-- 000003: Extend provision_tenant() agar membuat tabel investment_assets & unit_history
-- per tenant schema. Sebelumnya tabel ini diasumsikan ada oleh investment-service
-- tetapi tidak pernah di-DDL — menyebabkan crash pada operasi CRUD.
--
-- Pattern: provision_tenant() adalah single point of truth untuk DDL skema tenant di
-- kasku_finance. Investment-service tidak punya migration sendiri; ia bergantung pada
-- function ini untuk bootstrap tabel.

CREATE OR REPLACE FUNCTION public.provision_tenant(p_user_id UUID)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_schema TEXT;
BEGIN
    v_schema := 'tenant_' || replace(p_user_id::text, '-', '_');

    -- Buat schema jika belum ada
    EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', v_schema);

    -- Buat tabel financial_accounts
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

    -- Buat tabel transactions
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
            is_deleted        BOOLEAN      NOT NULL DEFAULT false,
            deleted_at        TIMESTAMPTZ  NULL,
            created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
            updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
        )', v_schema);

    -- Buat tabel categories
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

    -- Buat tabel balance_history (append-only audit log)
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.balance_history (
            id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
            account_id  UUID        NOT NULL,
            amount      BIGINT      NOT NULL,
            balance     BIGINT      NOT NULL,
            note        TEXT        NULL,
            created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
        )', v_schema);

    -- Buat tabel investment_assets (instrumen investasi per tenant)
    -- quantity: NUMERIC(28,8) untuk presisi crypto (max 8 decimals)
    -- avg_buy_price: NUMERIC(20,4) cukup untuk IDR & valuta asing dengan 4 desimal
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

    -- Index untuk query daftar aset aktif: WHERE is_deleted=false ORDER BY sort_order, created_at
    EXECUTE format('
        CREATE INDEX IF NOT EXISTS idx_investment_assets_active
        ON %I.investment_assets (sort_order ASC, created_at ASC)
        WHERE is_deleted = false', v_schema);

    -- Buat tabel unit_history (append-only riwayat BUY/SELL/ADJUST per aset)
    -- Tanpa FK eksplisit untuk konsistensi dengan pattern balance_history.
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

    -- Index untuk query history per aset terurut waktu (terbaru dulu)
    EXECUTE format('
        CREATE INDEX IF NOT EXISTS idx_unit_history_asset_recorded
        ON %I.unit_history (asset_id, recorded_at DESC)', v_schema);

    -- Seed default categories (idempotent via ON CONFLICT DO NOTHING)
    EXECUTE format('
        INSERT INTO %I.categories (name, icon, color, category_type, is_default) VALUES
            (''Gaji'',           ''briefcase'',      ''#22c55e'', ''INCOME'',  true),
            (''Bisnis'',         ''store'',           ''#16a34a'', ''INCOME'',  true),
            (''Investasi'',      ''trending-up'',     ''#15803d'', ''INCOME'',  true),
            (''Bonus'',          ''gift'',             ''#4ade80'', ''INCOME'',  true),
            (''Makanan'',        ''utensils'',         ''#f97316'', ''EXPENSE'', true),
            (''Transportasi'',   ''car'',              ''#3b82f6'', ''EXPENSE'', true),
            (''Belanja'',        ''shopping-bag'',     ''#a855f7'', ''EXPENSE'', true),
            (''Kesehatan'',      ''heart-pulse'',      ''#ef4444'', ''EXPENSE'', true),
            (''Tagihan'',        ''file-text'',        ''#f59e0b'', ''EXPENSE'', true),
            (''Hiburan'',        ''music'',            ''#ec4899'', ''EXPENSE'', true),
            (''Pendidikan'',     ''book-open'',        ''#06b6d4'', ''EXPENSE'', true),
            (''Tabungan'',       ''piggy-bank'',       ''#84cc16'', ''BOTH'',    true),
            (''Transfer'',       ''arrow-left-right'', ''#64748b'', ''BOTH'',    true),
            (''Lainnya'',        ''more-horizontal'',  ''#94a3b8'', ''BOTH'',    true)
        ON CONFLICT DO NOTHING', v_schema);

    -- Grant akses ke semua service yang membutuhkan kasku_finance
    EXECUTE format('GRANT USAGE ON SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
END;
$$;

-- Grant EXECUTE hanya ke service yang berhak melakukan provisioning
GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_finance_svc;

-- Backfill: create investment_assets & unit_history untuk tenant schemas yang
-- sudah ada (di-provision sebelum migration ini). CREATE TABLE IF NOT EXISTS
-- + CREATE INDEX IF NOT EXISTS membuat operasi idempotent.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^tenant_[0-9a-f_]+$'
    LOOP
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
            )', r.schema_name);

        EXECUTE format('
            CREATE INDEX IF NOT EXISTS idx_investment_assets_active
            ON %I.investment_assets (sort_order ASC, created_at ASC)
            WHERE is_deleted = false', r.schema_name);

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
            )', r.schema_name);

        EXECUTE format('
            CREATE INDEX IF NOT EXISTS idx_unit_history_asset_recorded
            ON %I.unit_history (asset_id, recorded_at DESC)', r.schema_name);

        -- Explicit GRANT pada tabel baru. ALTER DEFAULT PRIVILEGES sebelumnya
        -- hanya berlaku untuk tabel yang dibuat oleh role yang sama dengan
        -- yang men-set default privileges; backfill ini dijalankan oleh
        -- migration runner yang bisa berbeda, jadi grant eksplisit di sini.
        EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I.investment_assets, %I.unit_history TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', r.schema_name, r.schema_name);
    END LOOP;
END $$;
