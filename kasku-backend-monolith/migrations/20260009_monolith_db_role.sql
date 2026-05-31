-- 20260009: Single monolith DB role — kasku_app
-- Run as superuser before migrations 20260002–20260008 if role doesn't exist.
-- This migration is idempotent.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kasku_app') THEN
        CREATE ROLE kasku_app LOGIN PASSWORD 'changeme';
    END IF;
END $$;

-- Grant on all existing schemas
GRANT USAGE ON SCHEMA auth, user_mgmt, billing, notification, admin_panel, price TO kasku_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA auth TO kasku_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA user_mgmt TO kasku_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA billing TO kasku_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA notification TO kasku_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA admin_panel TO kasku_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA price TO kasku_app;

-- Default privileges for tables created in the future
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA user_mgmt GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA billing GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA notification GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA admin_panel GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA price GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kasku_app;

-- Sequences
GRANT USAGE ON ALL SEQUENCES IN SCHEMA auth TO kasku_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA user_mgmt TO kasku_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA billing TO kasku_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA notification TO kasku_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA admin_panel TO kasku_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA price TO kasku_app;

-- Execute provisioning functions
GRANT EXECUTE ON FUNCTION public.provision_tenant(UUID) TO kasku_app;
GRANT EXECUTE ON FUNCTION public.ensure_tenant_runtime_objects(TEXT) TO kasku_app;
