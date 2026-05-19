-- Migration: sync-service schema version registry
-- Tabel ini dimiliki oleh sync-service untuk tracking versi migrasinya sendiri.
-- Per-tenant sync_log dikelola oleh finance-service provision_tenant() function.
CREATE TABLE IF NOT EXISTS public.sync_service_schema_version (
    version     INTEGER     PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    description TEXT        NOT NULL
);

INSERT INTO public.sync_service_schema_version (version, description)
VALUES (1, 'initial sync-service metadata table')
ON CONFLICT (version) DO NOTHING;
