-- Migration: 001_create_price_cache
-- Database: kasku_price
-- Service: price-service

CREATE TABLE IF NOT EXISTS public.price_cache (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol      VARCHAR(20)     NOT NULL,
    source      VARCHAR(30)     NOT NULL CHECK (source IN ('COINGECKO', 'METALS_LIVE', 'MANUAL')),
    price_idr   NUMERIC(30, 8)  NOT NULL CHECK (price_idr > 0),
    price_usd   NUMERIC(30, 8)  NOT NULL CHECK (price_usd > 0),
    fetched_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ     NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS price_cache_symbol_source_unique_idx
    ON public.price_cache (symbol, source);

CREATE INDEX IF NOT EXISTS price_cache_expires_at_idx
    ON public.price_cache (expires_at);

COMMENT ON TABLE public.price_cache IS
    'TTL-based price cache. Default TTL: 900 seconds (15 minutes).';
COMMENT ON COLUMN public.price_cache.symbol IS
    'Asset symbol. Examples: bitcoin, ethereum, XAU (gold), XAG (silver).';
