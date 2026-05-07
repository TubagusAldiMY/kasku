CREATE TABLE IF NOT EXISTS public.users (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    email               VARCHAR(254)    NOT NULL,
    username            VARCHAR(30)     NOT NULL,
    password_hash       TEXT            NOT NULL,
    is_active           BOOLEAN         NOT NULL DEFAULT false,
    email_verified      BOOLEAN         NOT NULL DEFAULT false,
    failed_login_count  SMALLINT        NOT NULL DEFAULT 0
                            CHECK (failed_login_count >= 0 AND failed_login_count <= 10),
    locked_until        TIMESTAMPTZ     NULL,
    last_login_at       TIMESTAMPTZ     NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
    ON public.users (LOWER(email));

CREATE UNIQUE INDEX IF NOT EXISTS users_username_unique_idx
    ON public.users (LOWER(username));

CREATE INDEX IF NOT EXISTS users_is_active_idx
    ON public.users (is_active)
    WHERE is_active = true;

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON public.users
    FOR EACH ROW
    EXECUTE FUNCTION public.set_updated_at_timestamp();
