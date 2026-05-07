CREATE TABLE IF NOT EXISTS public.refresh_tokens (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID            NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)        NOT NULL,
    user_agent  VARCHAR(512)    NULL,
    ip_address  INET            NULL,
    expires_at  TIMESTAMPTZ     NOT NULL,
    is_revoked  BOOLEAN         NOT NULL DEFAULT false,
    revoked_at  TIMESTAMPTZ     NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS refresh_tokens_token_hash_unique_idx
    ON public.refresh_tokens (token_hash);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_active_idx
    ON public.refresh_tokens (user_id)
    WHERE is_revoked = false;

CREATE INDEX IF NOT EXISTS refresh_tokens_expires_at_idx
    ON public.refresh_tokens (expires_at)
    WHERE is_revoked = false;
