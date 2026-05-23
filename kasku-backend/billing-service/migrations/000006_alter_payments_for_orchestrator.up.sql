-- Migration: Alter tabel payments untuk Payment Orchestrator (menggantikan integrasi Midtrans)
--
-- Perubahan:
-- 1. Lepas FK constraint subscription_id (akan menjadi nullable — payment dibuat SEBELUM subscription)
-- 2. Rename midtrans_id → external_payment_id
-- 3. Tambah kolom: payment_method, payment_url, external_ref_id, expires_at
-- 4. Update status CHECK constraint: PENDING | PAID | FAILED | EXPIRED
-- 5. Tambah index pada external_ref_id dan status

-- Step 1: Buat tabel baru dengan skema yang benar
-- (DROP + RECREATE lebih aman daripada serangkaian ALTER di environment dev
--  karena migration 000004 belum pernah ada data production)
DROP TABLE IF EXISTS public.payments;

CREATE TABLE public.payments (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id     UUID         NULL REFERENCES public.subscriptions(id),
    user_id             UUID         NOT NULL,
    plan_id             UUID         NOT NULL REFERENCES public.subscription_plans(id),
    order_id            VARCHAR(150) NOT NULL UNIQUE,
    amount_idr          INTEGER      NOT NULL CHECK (amount_idr > 0),
    status              VARCHAR(20)  NOT NULL DEFAULT 'PENDING'
                            CHECK (status IN ('PENDING', 'PAID', 'FAILED', 'EXPIRED')),
    payment_method      VARCHAR(30)  NOT NULL DEFAULT 'QRIS'
                            CHECK (payment_method IN ('QRIS', 'VIRTUAL_ACCOUNT')),
    payment_url         TEXT         NULL,
    external_payment_id VARCHAR(150) NULL,
    external_ref_id     VARCHAR(150) NOT NULL,
    expires_at          TIMESTAMPTZ  NULL,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Index untuk lookup idempotency via external_ref_id (dari webhook callback orchestrator)
CREATE INDEX idx_payments_external_ref_id ON public.payments (external_ref_id);

-- Index untuk monitoring & cleanup pembayaran berdasarkan status
CREATE INDEX idx_payments_status ON public.payments (status);

-- Index untuk riwayat pembayaran per user
CREATE INDEX idx_payments_user_id ON public.payments (user_id);

-- Trigger auto-update updated_at
CREATE TRIGGER payments_set_updated_at
    BEFORE UPDATE ON public.payments
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();
