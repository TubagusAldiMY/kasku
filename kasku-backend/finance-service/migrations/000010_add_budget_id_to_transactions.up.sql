-- 000010: Allow each expense transaction to explicitly point at one budget.

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^tenant_[0-9a-f_]+$'
    LOOP
        EXECUTE format(
            'ALTER TABLE %I.transactions ADD COLUMN IF NOT EXISTS budget_id UUID NULL',
            r.schema_name
        );

        EXECUTE format(
            'CREATE INDEX IF NOT EXISTS transactions_budget_active ON %I.transactions (budget_id) WHERE is_deleted = false',
            r.schema_name
        );

        EXECUTE format(
            'UPDATE %1$I.transactions t
             SET budget_id = (
                 SELECT b.id
                 FROM %1$I.budgets b
                 JOIN %1$I.financial_accounts fa ON fa.user_id = b.user_id
                 WHERE fa.id = t.account_id
                   AND b.category_id IS NULL
                   AND b.is_deleted = false
                   AND t.transaction_date >= CASE b.period_type
                         WHEN ''MONTHLY'' THEN date_trunc(''month'', t.transaction_date)::date
                         WHEN ''WEEKLY'' THEN date_trunc(''week'', t.transaction_date)::date
                         ELSE b.start_date
                       END
                   AND t.transaction_date < CASE b.period_type
                         WHEN ''MONTHLY'' THEN (date_trunc(''month'', t.transaction_date) + interval ''1 month'')::date
                         WHEN ''WEEKLY'' THEN (date_trunc(''week'', t.transaction_date) + interval ''1 week'')::date
                         ELSE COALESCE(b.end_date + 1, t.transaction_date + 1)
                       END
                 ORDER BY b.created_at ASC
                 LIMIT 1
             )
             WHERE t.transaction_type = ''EXPENSE''
               AND t.budget_id IS NULL
               AND t.is_deleted = false',
            r.schema_name
        );
    END LOOP;
END $$;

-- User-service calls this right after provision_tenant(); keep it responsible
-- for tenant runtime drift so new tenants also get the budget_id column.
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

    EXECUTE format(
        'ALTER TABLE %I.transactions ADD COLUMN IF NOT EXISTS budget_id UUID NULL',
        p_tenant_schema
    );

    EXECUTE format(
        'CREATE INDEX IF NOT EXISTS transactions_budget_active ON %I.transactions (budget_id) WHERE is_deleted = false',
        p_tenant_schema
    );

    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I.sync_log TO kasku_sync_svc', p_tenant_schema);
END;
$$;

GRANT EXECUTE ON FUNCTION public.ensure_tenant_runtime_objects(TEXT) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION public.ensure_tenant_runtime_objects(TEXT) TO kasku_finance_svc;
