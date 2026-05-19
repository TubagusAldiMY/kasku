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

    -- Grant akses ke semua service yang membutuhkan kasku_finance
    EXECUTE format('GRANT USAGE ON SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
    EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_finance_svc, kasku_transaction_svc, kasku_investment_svc, kasku_sync_svc', v_schema);
END;
$$;

-- Grant EXECUTE hanya ke service yang berhak melakukan provisioning
GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_user_svc;
GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_finance_svc;
