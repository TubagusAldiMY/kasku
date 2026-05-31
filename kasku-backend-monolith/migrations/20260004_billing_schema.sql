-- 20260004: Billing schema — subscription_plans, subscriptions, payments, outbox

CREATE TABLE IF NOT EXISTS billing.subscription_plans (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(20) NOT NULL UNIQUE,
    price_idr  INTEGER     NOT NULL DEFAULT 0,
    limits     JSONB       NOT NULL DEFAULT '{}',
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO billing.subscription_plans (name, price_idr, limits) VALUES
(
    'FREE',
    0,
    '{
        "MaxTransactionsPerMonth": 50,
        "MaxFinancialAccounts": 3,
        "MaxInvestmentInstruments": 0,
        "HistoryRetentionMonths": 3,
        "EmailNotificationsEnabled": false,
        "ExportCsvEnabled": false
    }'
),
(
    'BASIC',
    49000,
    '{
        "MaxTransactionsPerMonth": 500,
        "MaxFinancialAccounts": 10,
        "MaxInvestmentInstruments": 5,
        "HistoryRetentionMonths": 12,
        "EmailNotificationsEnabled": true,
        "ExportCsvEnabled": true
    }'
),
(
    'PRO',
    99000,
    '{
        "MaxTransactionsPerMonth": -1,
        "MaxFinancialAccounts": -1,
        "MaxInvestmentInstruments": -1,
        "HistoryRetentionMonths": -1,
        "EmailNotificationsEnabled": true,
        "ExportCsvEnabled": true
    }'
)
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS billing.subscriptions (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID        NOT NULL UNIQUE,
    plan_id              UUID        NOT NULL REFERENCES billing.subscription_plans(id),
    status               VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT now(),
    current_period_end   TIMESTAMPTZ NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_status_period_end
    ON billing.subscriptions (status, current_period_end)
    WHERE current_period_end IS NOT NULL;

CREATE TRIGGER subscriptions_set_updated_at
    BEFORE UPDATE ON billing.subscriptions
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();

CREATE TABLE IF NOT EXISTS billing.payments (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    plan_id         UUID        NOT NULL REFERENCES billing.subscription_plans(id),
    order_id        VARCHAR(100) NOT NULL UNIQUE,
    amount_idr      BIGINT      NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    payment_method  VARCHAR(50) NULL,
    duration_days   INTEGER     NOT NULL DEFAULT 30,
    orchestrator_ref VARCHAR(200) NULL,
    paid_at         TIMESTAMPTZ NULL,
    expired_at      TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS payments_user_id_idx ON billing.payments (user_id);
CREATE INDEX IF NOT EXISTS payments_order_id_idx ON billing.payments (order_id);
CREATE INDEX IF NOT EXISTS payments_status_idx ON billing.payments (status) WHERE status = 'PENDING';

CREATE TRIGGER payments_set_updated_at
    BEFORE UPDATE ON billing.payments
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();

CREATE TABLE IF NOT EXISTS billing.outbox_events (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type       VARCHAR(100) NOT NULL,
    routing_key      VARCHAR(100) NOT NULL,
    payload          JSONB       NOT NULL,
    published_at     TIMESTAMPTZ NULL,
    publish_attempts INTEGER     NOT NULL DEFAULT 0,
    last_error       TEXT        NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS billing_outbox_unpublished_idx
    ON billing.outbox_events (created_at)
    WHERE published_at IS NULL;
