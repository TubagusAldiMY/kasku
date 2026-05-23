-- Rollback 000010: remove explicit budget assignment from transactions.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^tenant_[0-9a-f_]+$'
    LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I.transactions_budget_active', r.schema_name);
        EXECUTE format('ALTER TABLE %I.transactions DROP COLUMN IF EXISTS budget_id', r.schema_name);
    END LOOP;
END $$;

CREATE OR REPLACE FUNCTION public.ensure_tenant_runtime_objects(p_tenant_schema TEXT)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    IF p_tenant_schema !~ '^tenant_[0-9a-f_]{32,36}$' THEN
        RAISE EXCEPTION 'invalid tenant schema: %', p_tenant_schema;
    END IF;

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

    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I.sync_log TO kasku_sync_svc', p_tenant_schema);
END;
$$;

GRANT EXECUTE ON FUNCTION public.ensure_tenant_runtime_objects(TEXT) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION public.ensure_tenant_runtime_objects(TEXT) TO kasku_finance_svc;
