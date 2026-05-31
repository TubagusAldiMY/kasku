use std::sync::Arc;
use chrono::Utc;
use serde_json::json;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::billing::domain::{error::BillingError, repository::{SubscriptionPlanRepository, SubscriptionRepository}};

pub struct CancelSubscriptionUseCase {
    sub_repo: Arc<dyn SubscriptionRepository>,
    pool: PgPool,
}

impl CancelSubscriptionUseCase {
    pub fn new(sub_repo: Arc<dyn SubscriptionRepository>, pool: PgPool) -> Self {
        Self { sub_repo, pool }
    }

    pub async fn execute(&self, user_id: Uuid, plan_name: &str, email: &str) -> Result<(), BillingError> {
        let sub = self.sub_repo.find_by_user_id(user_id).await?
            .ok_or(BillingError::SubscriptionNotFound)?;

        if sub.status != "ACTIVE" {
            return Err(BillingError::Internal(anyhow::anyhow!("subscription tidak aktif")));
        }

        let now = Utc::now();

        let mut tx = self.pool.begin().await.map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;

        sqlx::query(
            "UPDATE billing.subscriptions SET status = 'CANCELLED', updated_at = $2 WHERE user_id = $1",
        )
        .bind(user_id)
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;

        let payload = json!({
            "user_id": user_id.to_string(),
            "email": email,
            "plan_name": plan_name,
            "cancelled_at": now.to_rfc3339(),
        });

        sqlx::query(
            "INSERT INTO billing.outbox_events (id, event_type, routing_key, payload, created_at) VALUES ($1, $2, $3, $4::jsonb, $5)",
        )
        .bind(Uuid::new_v4())
        .bind("subscription.cancelled")
        .bind("subscription.cancelled")
        .bind(serde_json::to_string(&payload).unwrap())
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;

        tx.commit().await.map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;

        Ok(())
    }
}
