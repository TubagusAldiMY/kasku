DROP INDEX IF EXISTS public.admin_audit_log_action_idx;
DROP INDEX IF EXISTS public.admin_audit_log_created_at_idx;
DROP INDEX IF EXISTS public.admin_audit_log_target_user_id_idx;
DROP INDEX IF EXISTS public.admin_audit_log_admin_id_idx;
DROP TABLE IF EXISTS public.admin_audit_log;
