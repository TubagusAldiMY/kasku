-- 20260002: Auth schema — users, tokens, outbox

CREATE TABLE IF NOT EXISTS auth.users (
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

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx ON auth.users (LOWER(email));
CREATE UNIQUE INDEX IF NOT EXISTS users_username_unique_idx ON auth.users (LOWER(username));
CREATE INDEX IF NOT EXISTS users_is_active_idx ON auth.users (is_active) WHERE is_active = true;

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();

CREATE TABLE IF NOT EXISTS auth.refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    user_agent  TEXT        NULL,
    ip_address  VARCHAR(45) NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    is_revoked  BOOLEAN     NOT NULL DEFAULT false,
    revoked_at  TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON auth.refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_expires_idx ON auth.refresh_tokens (expires_at) WHERE is_revoked = false;

CREATE TABLE IF NOT EXISTS auth.email_verifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS email_verifications_user_id_idx ON auth.email_verifications (user_id);

CREATE TABLE IF NOT EXISTS auth.password_reset_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS password_reset_tokens_user_id_idx ON auth.password_reset_tokens (user_id);

CREATE TABLE IF NOT EXISTS auth.outbox_events (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type       VARCHAR(100) NOT NULL,
    routing_key      VARCHAR(100) NOT NULL,
    payload          JSONB       NOT NULL,
    published_at     TIMESTAMPTZ NULL,
    publish_attempts INTEGER     NOT NULL DEFAULT 0,
    last_error       TEXT        NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS auth_outbox_unpublished_idx
    ON auth.outbox_events (created_at)
    WHERE published_at IS NULL;
