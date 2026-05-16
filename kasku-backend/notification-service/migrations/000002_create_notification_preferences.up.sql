CREATE TABLE IF NOT EXISTS public.notification_preferences (
    user_id                UUID        PRIMARY KEY,
    email_enabled          BOOLEAN     NOT NULL DEFAULT true,
    payment_alerts_enabled BOOLEAN     NOT NULL DEFAULT true,
    expiry_alerts_enabled  BOOLEAN     NOT NULL DEFAULT true,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER notification_preferences_set_updated_at
    BEFORE UPDATE ON public.notification_preferences
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();
