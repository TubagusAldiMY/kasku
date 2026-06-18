ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS google_id VARCHAR(100) NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_google_id_unique_idx
    ON public.users (google_id)
    WHERE google_id IS NOT NULL;
