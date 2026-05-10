use chrono::{DateTime, Utc};
use rust_decimal::Decimal;
use sqlx::PgPool;
use uuid::Uuid;

use crate::domain::entity::{PriceCache, PriceSource};
use crate::domain::error::DomainError;

/// Repository for price_cache table operations.
#[derive(Clone)]
pub struct PriceCacheRepository {
    pool: PgPool,
}

/// Row type for SQLx deserialization from the database.
#[derive(sqlx::FromRow)]
struct PriceCacheRow {
    id: Uuid,
    symbol: String,
    source: String,
    price_idr: Decimal,
    price_usd: Decimal,
    fetched_at: DateTime<Utc>,
    expires_at: DateTime<Utc>,
}

impl From<PriceCacheRow> for PriceCache {
    fn from(row: PriceCacheRow) -> Self {
        PriceCache {
            id: row.id,
            symbol: row.symbol,
            source: PriceSource::from_str(&row.source).unwrap_or(PriceSource::Manual),
            price_idr: row.price_idr,
            price_usd: row.price_usd,
            fetched_at: row.fetched_at,
            expires_at: row.expires_at,
        }
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
    ) -> Result<Option<PriceCache>, DomainError> {
        let row = sqlx::query_as::<_, PriceCacheRow>(
            "SELECT id, symbol, source, price_idr, price_usd, fetched_at, expires_at
             FROM public.price_cache
             WHERE symbol = $1 AND source = $2",
        )
        .bind(symbol)
        .bind(source.as_str())
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        Ok(row.map(PriceCache::from))
    }

    /// Get cached price by symbol (any source). Returns the most recently fetched one.
    pub async fn get_by_symbol(&self, symbol: &str) -> Result<Option<PriceCache>, DomainError> {
        let row = sqlx::query_as::<_, PriceCacheRow>(
            "SELECT id, symbol, source, price_idr, price_usd, fetched_at, expires_at
             FROM public.price_cache
             WHERE symbol = $1
             ORDER BY fetched_at DESC
             LIMIT 1",
        )
        .bind(symbol)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        Ok(row.map(PriceCache::from))
    }

    /// Upsert a price cache entry (INSERT ON CONFLICT UPDATE).
    pub async fn upsert(
        &self,
        symbol: &str,
        source: &PriceSource,
        price_idr: Decimal,
        price_usd: Decimal,
        cache_ttl_seconds: i64,
    ) -> Result<PriceCache, DomainError> {
        let row = sqlx::query_as::<_, PriceCacheRow>(
            "INSERT INTO public.price_cache (symbol, source, price_idr, price_usd, fetched_at, expires_at)
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
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        Ok(PriceCache::from(row))
    }
}
