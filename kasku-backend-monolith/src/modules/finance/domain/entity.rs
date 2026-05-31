use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct FinancialAccount {
    pub id: Uuid,
    pub user_id: Uuid,
    pub name: String,
    pub account_type: String,
    pub balance: i64,
    pub initial_balance: i64,
    pub currency: String,
    pub color: String,
    pub icon: String,
    pub is_default: bool,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct BalanceHistory {
    pub id: Uuid,
    pub account_id: Uuid,
    pub amount: i64,
    pub balance: i64,
    pub note: Option<String>,
    pub created_at: DateTime<Utc>,
}
