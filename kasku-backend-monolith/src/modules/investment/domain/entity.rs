use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::fmt;
use std::str::FromStr;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum AssetType {
    Crypto,
    Gold,
    Stock,
    MutualFund,
    Other,
}

impl fmt::Display for AssetType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AssetType::Crypto => write!(f, "CRYPTO"),
            AssetType::Gold => write!(f, "GOLD"),
            AssetType::Stock => write!(f, "STOCK"),
            AssetType::MutualFund => write!(f, "MUTUAL_FUND"),
            AssetType::Other => write!(f, "OTHER"),
        }
    }
}

impl FromStr for AssetType {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "CRYPTO" => Ok(AssetType::Crypto),
            "GOLD" => Ok(AssetType::Gold),
            "STOCK" => Ok(AssetType::Stock),
            "MUTUAL_FUND" => Ok(AssetType::MutualFund),
            "OTHER" => Ok(AssetType::Other),
            other => Err(format!("unknown asset type: {}", other)),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InvestmentAsset {
    pub id: Uuid,
    pub name: String,
    pub asset_type: AssetType,
    pub symbol: String,
    pub quantity: f64,
    pub avg_buy_price: f64,
    pub currency: String,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
    pub sort_order: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl<'r> sqlx::FromRow<'r, sqlx::postgres::PgRow> for InvestmentAsset {
    fn from_row(row: &'r sqlx::postgres::PgRow) -> Result<Self, sqlx::Error> {
        use sqlx::Row;
        let type_str: String = row.try_get("asset_type")?;
        let asset_type = type_str.parse::<AssetType>().map_err(|e| {
            sqlx::Error::ColumnDecode {
                index: "asset_type".to_string(),
                source: Box::new(std::io::Error::new(std::io::ErrorKind::InvalidData, e)),
            }
        })?;
        Ok(InvestmentAsset {
            id: row.try_get("id")?,
            name: row.try_get("name")?,
            asset_type,
            symbol: row.try_get("symbol")?,
            quantity: row.try_get("quantity")?,
            avg_buy_price: row.try_get("avg_buy_price")?,
            currency: row.try_get("currency")?,
            is_deleted: row.try_get("is_deleted")?,
            deleted_at: row.try_get("deleted_at")?,
            sort_order: row.try_get("sort_order")?,
            created_at: row.try_get("created_at")?,
            updated_at: row.try_get("updated_at")?,
        })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum UnitTransactionType {
    Buy,
    Sell,
    Adjust,
}

impl fmt::Display for UnitTransactionType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            UnitTransactionType::Buy => write!(f, "BUY"),
            UnitTransactionType::Sell => write!(f, "SELL"),
            UnitTransactionType::Adjust => write!(f, "ADJUST"),
        }
    }
}

impl FromStr for UnitTransactionType {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "BUY" => Ok(UnitTransactionType::Buy),
            "SELL" => Ok(UnitTransactionType::Sell),
            "ADJUST" => Ok(UnitTransactionType::Adjust),
            other => Err(format!("unknown unit transaction type: {}", other)),
        }
    }
}

#[derive(Debug, Clone)]
pub struct UnitHistory {
    pub id: Uuid,
    pub asset_id: Uuid,
    pub transaction_type: UnitTransactionType,
    pub quantity_change: f64,
    pub price_per_unit: f64,
    pub total_value: f64,
    pub notes: String,
    pub transaction_date: DateTime<Utc>,
    pub recorded_at: DateTime<Utc>,
}

impl<'r> sqlx::FromRow<'r, sqlx::postgres::PgRow> for UnitHistory {
    fn from_row(row: &'r sqlx::postgres::PgRow) -> Result<Self, sqlx::Error> {
        use sqlx::Row;
        let type_str: String = row.try_get("transaction_type")?;
        let transaction_type = type_str.parse::<UnitTransactionType>().map_err(|e| {
            sqlx::Error::ColumnDecode {
                index: "transaction_type".to_string(),
                source: Box::new(std::io::Error::new(std::io::ErrorKind::InvalidData, e)),
            }
        })?;
        Ok(UnitHistory {
            id: row.try_get("id")?,
            asset_id: row.try_get("asset_id")?,
            transaction_type,
            quantity_change: row.try_get("quantity_change")?,
            price_per_unit: row.try_get("price_per_unit")?,
            total_value: row.try_get("total_value")?,
            notes: row.try_get("notes")?,
            transaction_date: row.try_get("transaction_date")?,
            recorded_at: row.try_get("recorded_at")?,
        })
    }
}
