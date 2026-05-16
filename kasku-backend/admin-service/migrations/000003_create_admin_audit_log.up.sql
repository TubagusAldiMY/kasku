-- admin_audit_log mencatat setiap aksi admin yang menyentuh data user / subscription.
-- target_user_id nullable supaya bisa mencatat LOGIN/LOGOUT (non-target actions).
-- metadata JSONB menampung detail kontekstual: reason override, old/new tier, ip, user_agent.
-- success BOOLEAN supaya audit log tetap berguna saat mutation gagal (lihat trade-off di README).
CREATE TABLE IF NOT EXISTS public.admin_audit_log (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id        UUID        NOT NULL REFERENCES public.admin_users(id) ON DELETE RESTRICT,
    action          VARCHAR(50) NOT NULL,
    target_user_id  UUID        NULL,
    target_entity   VARCHAR(50) NULL,
    metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    success         BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS admin_audit_log_admin_id_idx
    ON public.admin_audit_log (admin_id);

CREATE INDEX IF NOT EXISTS admin_audit_log_target_user_id_idx
    ON public.admin_audit_log (target_user_id)
    WHERE target_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS admin_audit_log_created_at_idx
    ON public.admin_audit_log (created_at DESC);

CREATE INDEX IF NOT EXISTS admin_audit_log_action_idx
    ON public.admin_audit_log (action, created_at DESC);
