CREATE TABLE public.subscription_plans (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(20) NOT NULL UNIQUE,
    price_idr  INTEGER     NOT NULL DEFAULT 0,
    limits     JSONB       NOT NULL DEFAULT '{}',
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed data: tiga tier yang sesuai dengan ADR dan memory arsitektur KasKu
-- FREE tier: gratis selamanya sebagai entry point
INSERT INTO public.subscription_plans (name, price_idr, limits) VALUES
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
);
