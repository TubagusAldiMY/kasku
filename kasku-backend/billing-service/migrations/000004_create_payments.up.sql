CREATE TABLE public.payments (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID         NOT NULL REFERENCES public.subscriptions(id),
    user_id         UUID         NOT NULL,
    order_id        VARCHAR(100) NOT NULL UNIQUE,
    amount_idr      INTEGER      NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    midtrans_id     VARCHAR(100) NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Index untuk query riwayat pembayaran per user
CREATE INDEX idx_payments_user_id ON public.payments (user_id);

-- Index untuk idempotency check via order_id (unique sudah ada, ini untuk lookup eksplisit)
CREATE INDEX idx_payments_order_id ON public.payments (order_id);

-- Trigger untuk auto-update updated_at pada setiap UPDATE
CREATE TRIGGER payments_set_updated_at
    BEFORE UPDATE ON public.payments
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();
