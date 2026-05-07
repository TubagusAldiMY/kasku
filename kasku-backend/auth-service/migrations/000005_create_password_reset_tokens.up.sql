CREATE TABLE IF NOT EXISTS public.password_reset_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)    NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS password_reset_tokens_token_hash_idx
    ON public.password_reset_tokens (token_hash);

CREATE INDEX IF NOT EXISTS password_reset_tokens_user_id_unused_idx
    ON public.password_reset_tokens (user_id)
    WHERE used_at IS NULL;
