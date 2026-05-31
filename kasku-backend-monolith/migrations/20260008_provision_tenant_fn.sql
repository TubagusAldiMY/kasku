-- 20260008: provision_tenant() + ensure_tenant_runtime_objects()
-- Final version consolidated from finance-service migrations 000002–000010.
-- All per-service role grants replaced with kasku_app (single monolith role).

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
            balance         BIGINT       NOT NULL DEFAULT 0 CHECK (balance >= 0),
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
            budget_id         UUID         NULL,
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
        CREATE INDEX IF NOT EXISTS transactions_budget_active
        ON %I.transactions (budget_id) WHERE is_deleted = false', v_schema);

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
        CREATE UNIQUE INDEX IF NOT EXISTS categories_default_unique_idx
        ON %I.categories (name, category_type)
        WHERE is_default = true', v_schema);

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
            id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
            asset_id          UUID            NOT NULL,
            transaction_type  VARCHAR(20)     NOT NULL
                                CHECK (transaction_type IN (''BUY'',''SELL'',''ADJUST'')),
            quantity_change   NUMERIC(28, 8)  NOT NULL,
            price_per_unit    NUMERIC(20, 4)  NOT NULL,
            notes             TEXT            NULL,
            transaction_date  DATE            NOT NULL,
            recorded_at       TIMESTAMPTZ     NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS idx_unit_history_asset_recorded
        ON %I.unit_history (asset_id, recorded_at DESC)', v_schema);

    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.budgets (
            id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id         UUID         NOT NULL,
            sync_id         VARCHAR(100) UNIQUE,
            name            VARCHAR(100) NOT NULL,
            limit_idr       BIGINT       NOT NULL CHECK (limit_idr > 0),
            category_id     UUID         NULL,
            period_type     VARCHAR(20)  NOT NULL DEFAULT ''MONTHLY'',
            start_date      DATE         NOT NULL DEFAULT CURRENT_DATE,
            end_date        DATE         NULL,
            alert_threshold SMALLINT     NOT NULL DEFAULT 80 CHECK (alert_threshold BETWEEN 0 AND 100),
            is_deleted      BOOLEAN      NOT NULL DEFAULT false,
            deleted_at      TIMESTAMPTZ  NULL,
            created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
            updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
        )', v_schema);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS budgets_user_active
        ON %I.budgets (user_id) WHERE is_deleted = false', v_schema);

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
        )', v_schema);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS sync_log_entity_synced_at_idx
        ON %I.sync_log (entity_type, entity_id, synced_at DESC)', v_schema);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS sync_log_synced_at_idx
        ON %I.sync_log (synced_at DESC)', v_schema);

    EXECUTE format('GRANT USAGE ON SCHEMA %I TO kasku_app', v_schema);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_app', v_schema);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_app', v_schema);
END;
$$;

-- ensure_tenant_runtime_objects: idempotent drift correction for existing tenants
CREATE OR REPLACE FUNCTION public.ensure_tenant_runtime_objects(p_tenant_schema TEXT)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    IF p_tenant_schema !~ '^tenant_[0-9a-f_]{32,36}$' THEN
        RAISE EXCEPTION 'invalid tenant schema: %', p_tenant_schema;
    END IF;

    -- Ensure sync_log exists
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
        )', p_tenant_schema);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS sync_log_entity_synced_at_idx
        ON %I.sync_log (entity_type, entity_id, synced_at DESC)', p_tenant_schema);

    EXECUTE format('
        CREATE INDEX IF NOT EXISTS sync_log_synced_at_idx
        ON %I.sync_log (synced_at DESC)', p_tenant_schema);

    -- Deduplicate default categories before adding unique index
    EXECUTE format('
        DELETE FROM %I.categories c
        USING %I.categories d
        WHERE c.ctid < d.ctid
          AND c.is_default = true
          AND d.is_default = true
          AND c.name = d.name
          AND c.category_type = d.category_type', p_tenant_schema, p_tenant_schema);

    EXECUTE format('
        CREATE UNIQUE INDEX IF NOT EXISTS categories_default_unique_idx
        ON %I.categories (name, category_type)
        WHERE is_default = true', p_tenant_schema);

    -- Ensure budget_id column on transactions
    EXECUTE format(
        'ALTER TABLE %I.transactions ADD COLUMN IF NOT EXISTS budget_id UUID NULL',
        p_tenant_schema
    );

    EXECUTE format(
        'CREATE INDEX IF NOT EXISTS transactions_budget_active ON %I.transactions (budget_id) WHERE is_deleted = false',
        p_tenant_schema
    );

    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I.sync_log TO kasku_app', p_tenant_schema);
END;
$$;

GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_app;
GRANT EXECUTE ON FUNCTION public.ensure_tenant_runtime_objects(TEXT) TO kasku_app;
