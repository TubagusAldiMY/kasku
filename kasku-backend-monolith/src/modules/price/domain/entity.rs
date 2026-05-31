use chrono::{DateTime, Utc};
use rust_decimal::Decimal;
use serde::Serialize;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize)]
pub struct PriceCache {
    pub id: Uuid,
    pub symbol: String,
    pub source: PriceSource,
    pub price_idr: Decimal,
    pub price_usd: Decimal,
    pub fetched_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub enum PriceSource {
    MetalsLive,
    Manual,
}

impl PriceSource {
    pub fn as_str(&self) -> &str {
        match self {
            PriceSource::MetalsLive => "METALS_LIVE",
            PriceSource::Manual => "MANUAL",
        }
    }

    pub fn from_str(s: &str) -> Option<Self> {
        match s.to_uppercase().as_str() {
            "METALS_LIVE" => Some(PriceSource::MetalsLive),
            "MANUAL" => Some(PriceSource::Manual),
            _ => None,
        }
    }
}

impl std::fmt::Display for PriceSource {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct PriceResult {
    pub symbol: String,
    pub price_idr: f64,
    pub price_usd: f64,
    pub is_fresh: bool,
    pub updated_at: DateTime<Utc>,
}
