-- Constraint dilepas lebih dulu supaya DROP COLUMN tidak tertahan olehnya.
ALTER TABLE public.user_profiles
    DROP CONSTRAINT IF EXISTS user_profiles_avatar_size;
ALTER TABLE public.user_profiles
    DROP CONSTRAINT IF EXISTS user_profiles_avatar_complete;

ALTER TABLE public.user_profiles
    DROP COLUMN IF EXISTS avatar_updated_at,
    DROP COLUMN IF EXISTS avatar_mime,
    DROP COLUMN IF EXISTS avatar_image;
