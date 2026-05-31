use rust_decimal::Decimal;
use sqlx::Row;
use sqlx_postgres::{PgPool, PgRow};

use crate::modules::price::domain::entity::{PriceCache, PriceSource};
use crate::modules::price::domain::error::PriceError;

#[derive(Clone)]
pub struct PriceCacheRepository {
    pool: PgPool,
}

fn price_cache_from_row(row: PgRow) -> PriceCache {
    let source: String = row.get("source");
    PriceCache {
        id: row.get("id"),
        symbol: row.get("symbol"),
        source: PriceSource::from_str(&source).unwrap_or(PriceSource::Manual),
        price_idr: row.get("price_idr"),
        price_usd: row.get("price_usd"),
        fetched_at: row.get("fetched_at"),
        expires_at: row.get("expires_at"),
    }
}

impl PriceCacheRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// Get cached price by symbol and source. Returns None if not found.
    pub async fn get_by_symbol_source(
        &self,
        symbol: &str,
        source: &PriceSource,
    ) -> Result<Option<PriceCache>, PriceError> {
        let row = sqlx::query(
            "SELECT id, symbol, source, price_idr, price_usd, fetched_at, expires_at
             FROM price.price_cache
             WHERE symbol = $1 AND source = $2",
        )
        .bind(symbol)
        .bind(source.as_str())
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| PriceError::DatabaseError(e.to_string()))?;

        Ok(row.map(price_cache_from_row))
    }

    /// Get cached price by symbol (any source). Returns the most recently fetched one.
    pub async fn get_by_symbol(&self, symbol: &str) -> Result<Option<PriceCache>, PriceError> {
        let row = sqlx::query(
            "SELECT id, symbol, source, price_idr, price_usd, fetched_at, expires_at
             FROM price.price_cache
             WHERE symbol = $1
             ORDER BY fetched_at DESC
             LIMIT 1",
        )
        .bind(symbol)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| PriceError::DatabaseError(e.to_string()))?;

        Ok(row.map(price_cache_from_row))
    }

    /// Upsert a price cache entry (INSERT ON CONFLICT UPDATE).
    pub async fn upsert(
        &self,
        symbol: &str,
        source: &PriceSource,
        price_idr: Decimal,
        price_usd: Decimal,
        cache_ttl_seconds: i64,
    ) -> Result<PriceCache, PriceError> {
        let row = sqlx::query(
            "INSERT INTO price.price_cache (symbol, source, price_idr, price_usd, fetched_at, expires_at)
             VALUES ($1, $2, $3, $4, now(), now() + make_interval(secs => $5))
             ON CONFLICT (symbol, source)
             DO UPDATE SET
                 price_idr  = EXCLUDED.price_idr,
                 price_usd  = EXCLUDED.price_usd,
                 fetched_at = EXCLUDED.fetched_at,
                 expires_at = EXCLUDED.expires_at
             RETURNING id, symbol, source, price_idr, price_usd, fetched_at, expires_at",
        )
        .bind(symbol)
        .bind(source.as_str())
        .bind(price_idr)
        .bind(price_usd)
        .bind(cache_ttl_seconds as f64)
        .fetch_one(&self.pool)
        .await
        .map_err(|e| PriceError::DatabaseError(e.to_string()))?;

        Ok(price_cache_from_row(row))
    }
}
