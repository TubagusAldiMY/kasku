DROP INDEX IF EXISTS public.users_google_id_unique_idx;
ALTER TABLE public.users DROP COLUMN IF EXISTS google_id;
