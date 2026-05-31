-- 20260001: Shared infrastructure — extensions, utility function, schema creation

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Utility trigger function shared across all schemas
CREATE OR REPLACE FUNCTION public.set_updated_at_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Schema creation for all domains
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS user_mgmt;
CREATE SCHEMA IF NOT EXISTS billing;
CREATE SCHEMA IF NOT EXISTS notification;
CREATE SCHEMA IF NOT EXISTS admin_panel;
CREATE SCHEMA IF NOT EXISTS price;
