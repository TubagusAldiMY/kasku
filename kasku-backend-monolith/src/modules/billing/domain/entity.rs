use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct SubscriptionPlan {
    pub id: Uuid,
    pub name: String,
    pub price_idr: i32,
    pub limits: serde_json::Value,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Subscription {
    pub id: Uuid,
    pub user_id: Uuid,
    pub plan_id: Uuid,
    pub status: String,
    pub current_period_start: DateTime<Utc>,
    pub current_period_end: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Payment {
    pub id: Uuid,
    pub user_id: Uuid,
    pub plan_id: Uuid,
    pub order_id: String,
    pub amount_idr: i64,
    pub status: String,
    pub payment_method: Option<String>,
    pub duration_days: i32,
    pub orchestrator_ref: Option<String>,
    pub paid_at: Option<DateTime<Utc>>,
    pub expired_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}
