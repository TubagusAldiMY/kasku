-- Revert 000003: kembalikan provision_tenant() ke versi tanpa investment tables
-- dan drop investment_assets/unit_history di setiap tenant schema existing.

-- 1) Drop tabel investment per tenant schema (CASCADE untuk index/constraint)
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^tenant_[0-9a-f_]+$'
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I.unit_history CASCADE', r.schema_name);
        EXECUTE format('DROP TABLE IF EXISTS %I.investment_assets CASCADE', r.schema_name);
    END LOOP;
END $$;

-- 2) Restore provision_tenant() ke body sebelumnya (sama persis dengan 000002 up).
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

    EXECUTE format('GRANT USAGE ON SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
END;
$$;

GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_finance_svc;
