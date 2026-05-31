use serde::Serialize;
use crate::modules::price::domain::entity::PriceResult;

#[derive(Debug, Serialize)]
pub struct PriceResponse {
    pub symbol: String,
    pub price_idr: f64,
    pub price_usd: f64,
    pub is_fresh: bool,
    pub updated_at: String,
}

impl From<PriceResult> for PriceResponse {
    fn from(r: PriceResult) -> Self {
        Self {
            symbol: r.symbol,
            price_idr: r.price_idr,
            price_usd: r.price_usd,
            is_fresh: r.is_fresh,
            updated_at: r.updated_at.to_rfc3339(),
        }
    }
}
