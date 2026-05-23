use reqwest::Client;
use rust_decimal::Decimal;
use serde::Deserialize;
use std::collections::HashMap;
use std::time::Duration;
use tracing::{error, warn};
use url::Url;

use crate::domain::error::DomainError;

/// SSRF protection: only these domains are allowed for external requests.
const ALLOWED_DOMAINS: &[&str] = &["api.coingecko.com", "api.metals.live"];

/// CoinGecko price fetcher.
pub struct CoinGeckoClient {
    client: Client,
    api_key: String,
}

#[derive(Debug, Deserialize)]
struct CoinGeckoSimplePriceResponse {
    #[serde(flatten)]
    prices: HashMap<String, CoinGeckoPriceEntry>,
}

#[derive(Debug, Deserialize)]
struct CoinGeckoPriceEntry {
    usd: Option<f64>,
    idr: Option<f64>,
}

impl CoinGeckoClient {
    pub fn new(timeout_seconds: u64, api_key: String) -> Result<Self, DomainError> {
        let client = Client::builder()
            .timeout(Duration::from_secs(timeout_seconds))
            .user_agent("KasKu/1.0 price-service")
            .build()
            .map_err(|e| {
                DomainError::Internal(format!("gagal membuat HTTP client CoinGecko: {}", e))
            })?;

        Ok(Self { client, api_key })
    }

    /// Fetch price from CoinGecko API for a given coin_id (e.g., "bitcoin", "ethereum").
    /// Returns (price_usd, price_idr).
    pub async fn fetch_price(&self, coin_id: &str) -> Result<(Decimal, Decimal), DomainError> {
        let url = format!(
            "https://api.coingecko.com/api/v3/simple/price?ids={}&vs_currencies=usd,idr",
            coin_id
        );

        // SSRF protection
        validate_url_domain(&url)?;

        let mut request = self.client.get(&url);
        if !self.api_key.is_empty() {
            request = request.header("x-cg-demo-api-key", &self.api_key);
        }

        let response = request.send().await.map_err(|e| {
            error!(coin_id = coin_id, error = %e, "CoinGecko API request gagal");
            DomainError::ExternalApiFailed(format!("CoinGecko: {}", e))
        })?;

        if !response.status().is_success() {
            let status = response.status();
            warn!(coin_id = coin_id, status = %status, "CoinGecko API mengembalikan error");
            return Err(DomainError::ExternalApiFailed(format!(
                "CoinGecko returned HTTP {}",
                status
            )));
        }

        let data: CoinGeckoSimplePriceResponse = response.json().await.map_err(|e| {
            error!(coin_id = coin_id, error = %e, "gagal parse response CoinGecko");
            DomainError::ExternalApiFailed(format!("CoinGecko parse error: {}", e))
        })?;

        let entry = data
            .prices
            .get(coin_id)
            .ok_or_else(|| DomainError::PriceNotFound(coin_id.to_string()))?;

        let price_usd = entry
            .usd
            .ok_or_else(|| DomainError::PriceNotFound(format!("{}/USD", coin_id)))?;
        let price_idr = entry
            .idr
            .ok_or_else(|| DomainError::PriceNotFound(format!("{}/IDR", coin_id)))?;

        Ok((
            Decimal::try_from(price_usd)
                .map_err(|e| DomainError::Internal(format!("decimal conversion: {}", e)))?,
            Decimal::try_from(price_idr)
                .map_err(|e| DomainError::Internal(format!("decimal conversion: {}", e)))?,
        ))
    }
}

/// metals.live price fetcher for gold/precious metals.
pub struct MetalsLiveClient {
    client: Client,
    metals_live_url: String,
    gold_usd_idr_rate: f64,
}

#[derive(Debug, Deserialize)]
struct MetalsLiveSpot {
    gold: Option<f64>,
}

impl MetalsLiveClient {
    pub fn new(
        timeout_seconds: u64,
        metals_live_url: String,
        gold_usd_idr_rate: f64,
    ) -> Result<Self, DomainError> {
        let client = Client::builder()
            .timeout(Duration::from_secs(timeout_seconds))
            .user_agent("KasKu/1.0 price-service")
            .build()
            .map_err(|e| {
                DomainError::Internal(format!("gagal membuat HTTP client metals.live: {}", e))
            })?;

        Ok(Self {
            client,
            metals_live_url,
            gold_usd_idr_rate,
        })
    }

    /// Fetch gold price from metals.live. Returns (price_usd, price_idr) per troy ounce.
    pub async fn fetch_gold_price(&self) -> Result<(Decimal, Decimal), DomainError> {
        // SSRF protection
        validate_url_domain(&self.metals_live_url)?;

        let response = self
            .client
            .get(&self.metals_live_url)
            .send()
            .await
            .map_err(|e| {
                error!(error = %e, "metals.live API request gagal");
                DomainError::ExternalApiFailed(format!("metals.live: {}", e))
            })?;

        if !response.status().is_success() {
            let status = response.status();
            warn!(status = %status, "metals.live API mengembalikan error");
            return Err(DomainError::ExternalApiFailed(format!(
                "metals.live returned HTTP {}",
                status
            )));
        }

        // metals.live returns an array of spot prices
        let data: Vec<MetalsLiveSpot> = response.json().await.map_err(|e| {
            error!(error = %e, "gagal parse response metals.live");
            DomainError::ExternalApiFailed(format!("metals.live parse error: {}", e))
        })?;

        let gold_usd = data
            .first()
            .and_then(|s| s.gold)
            .ok_or_else(|| DomainError::PriceNotFound("XAU".to_string()))?;

        let gold_idr = gold_usd * self.gold_usd_idr_rate;

        Ok((
            Decimal::try_from(gold_usd)
                .map_err(|e| DomainError::Internal(format!("decimal conversion: {}", e)))?,
            Decimal::try_from(gold_idr)
                .map_err(|e| DomainError::Internal(format!("decimal conversion: {}", e)))?,
        ))
    }
}

/// SSRF protection: validate that the URL domain is in the allowed whitelist.
fn validate_url_domain(url_str: &str) -> Result<(), DomainError> {
    let parsed = Url::parse(url_str)
        .map_err(|e| DomainError::SsrfBlocked(format!("URL tidak valid: {}", e)))?;

    let host = parsed
        .host_str()
        .ok_or_else(|| DomainError::SsrfBlocked("URL tidak memiliki host".to_string()))?;

    if !ALLOWED_DOMAINS.contains(&host) {
        return Err(DomainError::SsrfBlocked(format!(
            "domain '{}' tidak diizinkan (whitelist: {:?})",
            host, ALLOWED_DOMAINS
        )));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validate_url_domain_allowed() {
        assert!(validate_url_domain("https://api.coingecko.com/api/v3/simple/price").is_ok());
        assert!(validate_url_domain("https://api.metals.live/v1/spot/gold").is_ok());
    }

    #[test]
    fn test_validate_url_domain_blocked() {
        assert!(validate_url_domain("https://evil.com/steal-data").is_err());
        assert!(validate_url_domain("http://localhost:8080/admin").is_err());
        assert!(validate_url_domain("http://169.254.169.254/metadata").is_err());
    }
}
