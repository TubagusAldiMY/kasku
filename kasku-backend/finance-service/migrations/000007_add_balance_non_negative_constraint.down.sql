-- Rollback 000007: Remove CHECK (balance >= 0) from all tenant schemas
-- and restore provision_tenant to the 000006 version.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^tenant_[0-9a-f_]+$'
    LOOP
        BEGIN
            EXECUTE format(
                'ALTER TABLE %I.financial_accounts DROP CONSTRAINT IF EXISTS financial_accounts_balance_non_negative',
                r.schema_name
            );
        EXCEPTION
            WHEN undefined_object THEN NULL;
        END;
    END LOOP;
END $$;

-- Restore provision_tenant to 000006 version (without balance CHECK constraint).
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
            id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id         UUID         NOT NULL,
            name            VARCHAR(100) NOT NULL,
            account_type    VARCHAR(20)  NOT NULL DEFAULT ''BANK'',
            balance         BIGINT       NOT NULL DEFAULT 0,
            initial_balance BIGINT       NOT NULL DEFAULT 0,
            currency        CHAR(3)      NOT NULL DEFAULT ''IDR'',
            color           VARCHAR(7)   NOT NULL DEFAULT ''#6366f1'',
            icon            VARCHAR(50)  NOT NULL DEFAULT ''wallet'',
            is_default      BOOLEAN      NOT NULL DEFAULT false,
            is_deleted      BOOLEAN      NOT NULL DEFAULT false,
            deleted_at      TIMESTAMPTZ  NULL,
            created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
            updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
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
END;
$$;

GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_finance_svc;
