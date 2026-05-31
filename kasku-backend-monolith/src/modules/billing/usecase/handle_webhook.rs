use std::sync::Arc;
use chrono::Utc;
use hmac::{Hmac, Mac};
use sha2::Sha256;
use serde_json::json;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::billing::domain::{error::BillingError, repository::{PaymentRepository, SubscriptionRepository}};

pub struct WebhookPayload {
    pub order_id: String,
    pub status: String,
    pub signature: String,
    pub user_email: Option<String>,
}

pub struct HandleWebhookUseCase {
    payment_repo: Arc<dyn PaymentRepository>,
    sub_repo: Arc<dyn SubscriptionRepository>,
    pool: PgPool,
    webhook_secret: String,
}

impl HandleWebhookUseCase {
    pub fn new(
        payment_repo: Arc<dyn PaymentRepository>,
        sub_repo: Arc<dyn SubscriptionRepository>,
        pool: PgPool,
        webhook_secret: String,
    ) -> Self {
        Self { payment_repo, sub_repo, pool, webhook_secret }
    }

    pub async fn execute(&self, payload: WebhookPayload) -> Result<(), BillingError> {
        // HMAC-SHA256 verification
        let expected = self.compute_signature(&payload.order_id, &payload.status);
        if !constant_time_eq(expected.as_bytes(), payload.signature.as_bytes()) {
            return Err(BillingError::InvalidWebhookSignature);
        }

        let payment = self.payment_repo.find_by_order_id(&payload.order_id).await?
            .ok_or(BillingError::PaymentNotFound)?;

        if payment.status != "PENDING" {
            return Err(BillingError::PaymentAlreadyProcessed);
        }

        let now = Utc::now();

        let mut tx = self.pool.begin().await.map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;

        if payload.status == "SUCCESS" || payload.status == "PAID" {
            sqlx::query(
                "UPDATE billing.payments SET status = 'PAID', paid_at = $2, updated_at = $3 WHERE order_id = $1",
            )
            .bind(&payload.order_id)
            .bind(now)
            .bind(now)
            .execute(&mut *tx)
            .await
            .map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;

            let period_end = now + chrono::Duration::days(payment.duration_days as i64);
            sqlx::query(
                "UPDATE billing.subscriptions SET plan_id = $2, status = 'ACTIVE', current_period_start = $3, current_period_end = $4, updated_at = $3 WHERE user_id = $1",
            )
            .bind(payment.user_id)
            .bind(payment.plan_id)
            .bind(now)
            .bind(period_end)
            .execute(&mut *tx)
            .await
            .map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;

            // Outbox event for notification
            if let Some(email) = &payload.user_email {
                let ev_payload = json!({
                    "user_id": payment.user_id.to_string(),
                    "email": email,
                    "order_id": payment.order_id,
                    "amount_idr": payment.amount_idr,
                    "plan_name": "",
                });
                sqlx::query(
                    "INSERT INTO billing.outbox_events (id, event_type, routing_key, payload, created_at) VALUES ($1, $2, $3, $4::jsonb, $5)",
                )
                .bind(Uuid::new_v4())
                .bind("payment.succeeded")
                .bind("payment.succeeded")
                .bind(serde_json::to_string(&ev_payload).unwrap())
                .bind(now)
                .execute(&mut *tx)
                .await
                .map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;
            }
        } else {
            sqlx::query(
                "UPDATE billing.payments SET status = 'FAILED', updated_at = $2 WHERE order_id = $1",
            )
            .bind(&payload.order_id)
            .bind(now)
            .execute(&mut *tx)
            .await
            .map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;
        }

        tx.commit().await.map_err(|e| BillingError::Internal(anyhow::anyhow!(e)))?;

        Ok(())
    }

    fn compute_signature(&self, order_id: &str, status: &str) -> String {
        let mut mac = Hmac::<Sha256>::new_from_slice(self.webhook_secret.as_bytes()).unwrap();
        mac.update(format!("{}:{}", order_id, status).as_bytes());
        hex::encode(mac.finalize().into_bytes())
    }
}

fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() { return false; }
    a.iter().zip(b.iter()).fold(0u8, |acc, (x, y)| acc | (x ^ y)) == 0
}
