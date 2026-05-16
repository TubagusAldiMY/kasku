DROP TRIGGER IF EXISTS admin_users_set_updated_at ON public.admin_users;
DROP INDEX IF EXISTS public.admin_users_active_role_idx;
DROP INDEX IF EXISTS public.admin_users_username_unique_idx;
DROP TABLE IF EXISTS public.admin_users;
