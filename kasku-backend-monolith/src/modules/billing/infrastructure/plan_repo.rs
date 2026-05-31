use async_trait::async_trait;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::billing::domain::{
    entity::SubscriptionPlan,
    error::BillingError,
    repository::SubscriptionPlanRepository,
};

pub struct PostgresSubscriptionPlanRepository {
    pool: PgPool,
}

impl PostgresSubscriptionPlanRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl SubscriptionPlanRepository for PostgresSubscriptionPlanRepository {
    async fn list_active(&self) -> Result<Vec<SubscriptionPlan>, BillingError> {
        let rows = sqlx::query_as::<_, SubscriptionPlan>(
            "SELECT id, name, price_idr, limits, is_active, created_at FROM billing.subscription_plans WHERE is_active = true ORDER BY price_idr ASC",
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    async fn find_by_id(&self, id: Uuid) -> Result<Option<SubscriptionPlan>, BillingError> {
        let row = sqlx::query_as::<_, SubscriptionPlan>(
            "SELECT id, name, price_idr, limits, is_active, created_at FROM billing.subscription_plans WHERE id = $1",
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn find_by_name(&self, name: &str) -> Result<Option<SubscriptionPlan>, BillingError> {
        let row = sqlx::query_as::<_, SubscriptionPlan>(
            "SELECT id, name, price_idr, limits, is_active, created_at FROM billing.subscription_plans WHERE name = $1",
        )
        .bind(name)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }
}
