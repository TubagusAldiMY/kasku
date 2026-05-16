CREATE TABLE IF NOT EXISTS public.admin_users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(30) NOT NULL,
    password_hash   TEXT        NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'SUPPORT'
                        CHECK (role IN ('SUPER_ADMIN', 'SUPPORT')),
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS admin_users_username_unique_idx
    ON public.admin_users (LOWER(username));

CREATE INDEX IF NOT EXISTS admin_users_active_role_idx
    ON public.admin_users (is_active, role)
    WHERE is_active = true;

CREATE TRIGGER admin_users_set_updated_at
    BEFORE UPDATE ON public.admin_users
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
