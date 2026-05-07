CREATE TABLE IF NOT EXISTS public.email_verifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash  CHAR(64)    NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS email_verifications_token_hash_idx
    ON public.email_verifications (token_hash);

CREATE INDEX IF NOT EXISTS email_verifications_user_id_unverified_idx
    ON public.email_verifications (user_id)
    WHERE verified_at IS NULL;
