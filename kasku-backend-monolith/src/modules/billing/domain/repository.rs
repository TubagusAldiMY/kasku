use async_trait::async_trait;
use uuid::Uuid;

use super::entity::{Payment, Subscription, SubscriptionPlan};
use super::error::BillingError;

#[async_trait]
pub trait SubscriptionPlanRepository: Send + Sync {
    async fn list_active(&self) -> Result<Vec<SubscriptionPlan>, BillingError>;
    async fn find_by_id(&self, id: Uuid) -> Result<Option<SubscriptionPlan>, BillingError>;
    async fn find_by_name(&self, name: &str) -> Result<Option<SubscriptionPlan>, BillingError>;
}

#[async_trait]
pub trait SubscriptionRepository: Send + Sync {
    async fn find_by_user_id(&self, user_id: Uuid) -> Result<Option<Subscription>, BillingError>;
    async fn create(&self, sub: &Subscription) -> Result<(), BillingError>;
    async fn update_plan(&self, user_id: Uuid, plan_id: Uuid, period_end: Option<chrono::DateTime<chrono::Utc>>, status: &str) -> Result<(), BillingError>;
    async fn find_expiring(&self, within_hours: i64) -> Result<Vec<(Subscription, String)>, BillingError>;
    async fn expire_overdue(&self) -> Result<u64, BillingError>;
}

#[async_trait]
pub trait PaymentRepository: Send + Sync {
    async fn create(&self, payment: &Payment) -> Result<(), BillingError>;
    async fn find_by_order_id(&self, order_id: &str) -> Result<Option<Payment>, BillingError>;
    async fn find_by_user_id(&self, user_id: Uuid, page: i64, limit: i64) -> Result<Vec<Payment>, BillingError>;
    async fn list_all(&self, page: i64, limit: i64) -> Result<Vec<Payment>, BillingError>;
    async fn update_status(&self, order_id: &str, status: &str, paid_at: Option<chrono::DateTime<chrono::Utc>>) -> Result<(), BillingError>;
}
