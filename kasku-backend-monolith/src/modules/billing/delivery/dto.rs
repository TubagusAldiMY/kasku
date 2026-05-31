use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize)]
pub struct PlanResponse {
    pub id: String,
    pub name: String,
    pub price_idr: i32,
    pub limits: serde_json::Value,
}

#[derive(Debug, Serialize)]
pub struct SubscriptionResponse {
    pub id: String,
    pub status: String,
    pub plan: PlanResponse,
    pub current_period_start: String,
    pub current_period_end: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CreatePaymentRequest {
    pub plan_id: String,
}

#[derive(Debug, Serialize)]
pub struct CreatePaymentResponse {
    pub order_id: String,
    pub amount_idr: i64,
    pub payment_url: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct WebhookRequest {
    pub order_id: String,
    pub status: String,
    pub signature: String,
    pub user_email: Option<String>,
}
