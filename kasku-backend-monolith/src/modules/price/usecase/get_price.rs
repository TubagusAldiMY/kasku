use chrono::Utc;
use rust_decimal::prelude::ToPrimitive;
use tracing::{info, warn};

use crate::modules::price::domain::entity::{PriceResult, PriceSource};
use crate::modules::price::domain::error::PriceError;
use crate::modules::price::infrastructure::repository::PriceCacheRepository;
use crate::modules::price::usecase::fetch_external::MetalsLiveClient;

/// Use case: Get price for a symbol, using cache-first strategy.
///
/// 1. Check price_cache table — if valid (expires_at > now), return cached + is_fresh=true
/// 2. If expired or missing — call external API
/// 3. On success → UPSERT into cache, return fresh data
/// 4. On failure → return stale cached data with is_fresh=false (graceful fallback)
pub struct GetPriceUseCase {
    repo: PriceCacheRepository,
    metals_live: MetalsLiveClient,
    cache_ttl_seconds: u64,
}

impl GetPriceUseCase {
    pub fn new(
        repo: PriceCacheRepository,
        metals_live: MetalsLiveClient,
        cache_ttl_seconds: u64,
    ) -> Self {
        Self {
            repo,
            metals_live,
            cache_ttl_seconds,
        }
    }

    /// Get price for a single symbol.
    /// `source_hint` can be "METALS_LIVE" or empty for auto-detect.
    pub async fn execute(
        &self,
        symbol: &str,
        source_hint: &str,
    ) -> Result<PriceResult, PriceError> {
        let source = self.resolve_source(symbol, source_hint)?;

        // Step 1: Check cache
        if let Some(cached) = self.repo.get_by_symbol_source(symbol, &source).await? {
            if cached.expires_at > Utc::now() {
                return Ok(PriceResult {
                    symbol: symbol.to_string(),
                    price_idr: cached.price_idr.to_f64().unwrap_or(0.0),
                    price_usd: cached.price_usd.to_f64().unwrap_or(0.0),
                    is_fresh: true,
                    updated_at: cached.fetched_at,
                });
            }
        }

        // Step 2: Fetch from external API
        let fetch_result = self.metals_live.fetch_gold_price().await;

        match fetch_result {
            Ok((price_usd, price_idr)) => {
                // Step 3: Upsert into cache
                let cached = self
                    .repo
                    .upsert(
                        symbol,
                        &source,
                        price_idr,
                        price_usd,
                        self.cache_ttl_seconds as i64,
                    )
                    .await?;

                info!(symbol = symbol, source = %source, "harga berhasil di-refresh dari API");

                Ok(PriceResult {
                    symbol: symbol.to_string(),
                    price_idr: cached.price_idr.to_f64().unwrap_or(0.0),
                    price_usd: cached.price_usd.to_f64().unwrap_or(0.0),
                    is_fresh: true,
                    updated_at: cached.fetched_at,
                })
            }
            Err(err) => {
                // Step 4: Graceful fallback — return stale data if available
                warn!(symbol = symbol, error = %err, "gagal fetch harga dari API, mencoba fallback ke cache stale");

                if let Some(stale) = self.repo.get_by_symbol_source(symbol, &source).await? {
                    Ok(PriceResult {
                        symbol: symbol.to_string(),
                        price_idr: stale.price_idr.to_f64().unwrap_or(0.0),
                        price_usd: stale.price_usd.to_f64().unwrap_or(0.0),
                        is_fresh: false,
                        updated_at: stale.fetched_at,
                    })
                } else {
                    Err(PriceError::PriceNotFound(symbol.to_string()))
                }
            }
        }
    }

    /// Resolve the price source based on symbol and hint.
    /// Only XAU/XAG/GOLD/SILVER are supported (MetalsLive). Others are unsupported.
    fn resolve_source(
        &self,
        symbol: &str,
        source_hint: &str,
    ) -> Result<PriceSource, PriceError> {
        if !source_hint.is_empty() {
            if let Some(source) = PriceSource::from_str(source_hint) {
                return Ok(source);
            }
        }

        // Auto-detect based on symbol
        let upper = symbol.to_uppercase();
        if upper == "XAU" || upper == "XAG" || upper == "GOLD" || upper == "SILVER" {
            Ok(PriceSource::MetalsLive)
        } else {
            Err(PriceError::UnsupportedSource(symbol.to_string()))
        }
    }
}
