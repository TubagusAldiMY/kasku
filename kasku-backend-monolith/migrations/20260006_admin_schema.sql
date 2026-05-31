-- 20260006: Admin schema — admin_users, admin_audit_log

CREATE TABLE IF NOT EXISTS admin_panel.admin_users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(30) NOT NULL,
    password_hash TEXT        NOT NULL,
    role          VARCHAR(20) NOT NULL DEFAULT 'SUPPORT'
                      CHECK (role IN ('SUPER_ADMIN', 'SUPPORT')),
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_users_username_unique_idx
    ON admin_panel.admin_users (LOWER(username));

CREATE INDEX IF NOT EXISTS admin_users_active_role_idx
    ON admin_panel.admin_users (is_active, role)
    WHERE is_active = true;

CREATE TRIGGER admin_users_set_updated_at
    BEFORE UPDATE ON admin_panel.admin_users
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();

CREATE TABLE IF NOT EXISTS admin_panel.admin_audit_log (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id       UUID        NOT NULL REFERENCES admin_panel.admin_users(id) ON DELETE RESTRICT,
    action         VARCHAR(50) NOT NULL,
    target_user_id UUID        NULL,
    target_entity  VARCHAR(50) NULL,
    metadata       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    success        BOOLEAN     NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS admin_audit_log_admin_id_idx ON admin_panel.admin_audit_log (admin_id);
CREATE INDEX IF NOT EXISTS admin_audit_log_target_user_id_idx ON admin_panel.admin_audit_log (target_user_id);
CREATE INDEX IF NOT EXISTS admin_audit_log_action_idx ON admin_panel.admin_audit_log (action);
CREATE INDEX IF NOT EXISTS admin_audit_log_created_at_idx ON admin_panel.admin_audit_log (created_at DESC);
