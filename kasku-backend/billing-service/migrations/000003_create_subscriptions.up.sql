CREATE TABLE public.subscriptions (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID        NOT NULL UNIQUE,
    plan_id              UUID        NOT NULL REFERENCES public.subscription_plans(id),
    status               VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT now(),
    current_period_end   TIMESTAMPTZ NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index parsial untuk optimasi query expiry check background job
-- Hanya mengindeks baris yang memiliki current_period_end (subscription berbayar)
CREATE INDEX idx_subscriptions_status_period_end
    ON public.subscriptions (status, current_period_end)
    WHERE current_period_end IS NOT NULL;

-- Trigger untuk auto-update updated_at pada setiap UPDATE
CREATE TRIGGER subscriptions_set_updated_at
    BEFORE UPDATE ON public.subscriptions
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();

-- Grant akses untuk user-service agar bisa membaca subscription saat provisioning tenant
GRANT SELECT, INSERT, UPDATE ON public.subscriptions TO kasku_user_svc;
GRANT SELECT ON public.subscription_plans TO kasku_user_svc;
