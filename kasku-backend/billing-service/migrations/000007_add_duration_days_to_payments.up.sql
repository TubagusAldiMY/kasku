ALTER TABLE public.payments
    ADD COLUMN duration_days INTEGER NOT NULL DEFAULT 30 CHECK (duration_days > 0);
