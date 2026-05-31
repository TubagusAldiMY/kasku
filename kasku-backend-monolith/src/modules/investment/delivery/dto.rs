use serde::{Deserialize, Serialize};
use crate::modules::investment::domain::entity::{InvestmentAsset, UnitHistory};

#[derive(Debug, Serialize)]
pub struct AssetResponse {
    pub id: String,
    pub name: String,
    pub asset_type: String,
    pub symbol: String,
    pub quantity: f64,
    pub avg_buy_price: f64,
    pub currency: String,
    pub sort_order: i32,
    pub created_at: String,
    pub updated_at: String,
}

impl From<InvestmentAsset> for AssetResponse {
    fn from(a: InvestmentAsset) -> Self {
        Self {
            id: a.id.to_string(),
            name: a.name,
            asset_type: a.asset_type.to_string(),
            symbol: a.symbol,
            quantity: a.quantity,
            avg_buy_price: a.avg_buy_price,
            currency: a.currency,
            sort_order: a.sort_order,
            created_at: a.created_at.to_rfc3339(),
            updated_at: a.updated_at.to_rfc3339(),
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct CreateAssetRequest {
    pub name: String,
    #[serde(default = "default_asset_type")]
    pub asset_type: String,
    #[serde(default)]
    pub symbol: String,
    #[serde(default)]
    pub quantity: f64,
    #[serde(default)]
    pub avg_buy_price: f64,
    #[serde(default = "default_currency")]
    pub currency: String,
    #[serde(default)]
    pub sort_order: i32,
}

fn default_asset_type() -> String { "OTHER".into() }
fn default_currency() -> String { "IDR".into() }

#[derive(Debug, Deserialize)]
pub struct UpdateAssetRequest {
    pub name: Option<String>,
    pub asset_type: Option<String>,
    pub symbol: Option<String>,
    pub quantity: Option<f64>,
    pub avg_buy_price: Option<f64>,
    pub currency: Option<String>,
    pub sort_order: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct RecordTransactionRequest {
    pub transaction_type: String,
    pub quantity_change: f64,
    pub price_per_unit: f64,
    #[serde(default)]
    pub notes: String,
    pub transaction_date: String,
}

#[derive(Debug, Serialize)]
pub struct UnitHistoryResponse {
    pub id: String,
    pub asset_id: String,
    pub transaction_type: String,
    pub quantity_change: f64,
    pub price_per_unit: f64,
    pub total_value: f64,
    pub notes: String,
    pub transaction_date: String,
    pub recorded_at: String,
}

impl From<UnitHistory> for UnitHistoryResponse {
    fn from(h: UnitHistory) -> Self {
        Self {
            id: h.id.to_string(),
            asset_id: h.asset_id.to_string(),
            transaction_type: h.transaction_type.to_string(),
            quantity_change: h.quantity_change,
            price_per_unit: h.price_per_unit,
            total_value: h.total_value,
            notes: h.notes,
            transaction_date: h.transaction_date.to_rfc3339(),
            recorded_at: h.recorded_at.to_rfc3339(),
        }
    }
}
