-- 20260007: Price schema — price_cache (from price-service migration 001)
-- CoinGecko removed; only MetalsLive (XAU, XAG) sources used.

CREATE TABLE IF NOT EXISTS price.price_cache (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol      VARCHAR(20)     NOT NULL,
    source      VARCHAR(30)     NOT NULL CHECK (source IN ('METALS_LIVE', 'MANUAL')),
    price_idr   NUMERIC(30, 8)  NOT NULL CHECK (price_idr > 0),
    price_usd   NUMERIC(30, 8)  NOT NULL CHECK (price_usd > 0),
    fetched_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ     NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS price_cache_symbol_source_unique_idx
    ON price.price_cache (symbol, source);

CREATE INDEX IF NOT EXISTS price_cache_expires_at_idx
    ON price.price_cache (expires_at);
