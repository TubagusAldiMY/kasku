use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::billing::domain::{
    entity::Payment,
    error::BillingError,
    repository::PaymentRepository,
};

pub struct PostgresPaymentRepository {
    pool: PgPool,
}

impl PostgresPaymentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl PaymentRepository for PostgresPaymentRepository {
    async fn create(&self, p: &Payment) -> Result<(), BillingError> {
        sqlx::query(
            r#"INSERT INTO billing.payments
               (id, user_id, plan_id, order_id, amount_idr, status, payment_method, duration_days,
                orchestrator_ref, paid_at, expired_at, created_at, updated_at)
               VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)"#,
        )
        .bind(p.id)
        .bind(p.user_id)
        .bind(p.plan_id)
        .bind(&p.order_id)
        .bind(p.amount_idr)
        .bind(&p.status)
        .bind(&p.payment_method)
        .bind(p.duration_days)
        .bind(&p.orchestrator_ref)
        .bind(p.paid_at)
        .bind(p.expired_at)
        .bind(p.created_at)
        .bind(p.updated_at)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn find_by_order_id(&self, order_id: &str) -> Result<Option<Payment>, BillingError> {
        let row = sqlx::query_as::<_, Payment>(
            r#"SELECT id, user_id, plan_id, order_id, amount_idr, status, payment_method,
               duration_days, orchestrator_ref, paid_at, expired_at, created_at, updated_at
               FROM billing.payments WHERE order_id = $1 LIMIT 1"#,
        )
        .bind(order_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn find_by_user_id(&self, user_id: Uuid, page: i64, limit: i64) -> Result<Vec<Payment>, BillingError> {
        let offset = (page - 1) * limit;
        let rows = sqlx::query_as::<_, Payment>(
            r#"SELECT id, user_id, plan_id, order_id, amount_idr, status, payment_method,
               duration_days, orchestrator_ref, paid_at, expired_at, created_at, updated_at
               FROM billing.payments WHERE user_id = $1
               ORDER BY created_at DESC LIMIT $2 OFFSET $3"#,
        )
        .bind(user_id)
        .bind(limit)
        .bind(offset)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    async fn list_all(&self, page: i64, limit: i64) -> Result<Vec<Payment>, BillingError> {
        let offset = (page - 1) * limit;
        let rows = sqlx::query_as::<_, Payment>(
            r#"SELECT id, user_id, plan_id, order_id, amount_idr, status, payment_method,
               duration_days, orchestrator_ref, paid_at, expired_at, created_at, updated_at
               FROM billing.payments
               ORDER BY created_at DESC LIMIT $1 OFFSET $2"#,
        )
        .bind(limit)
        .bind(offset)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    async fn update_status(&self, order_id: &str, status: &str, paid_at: Option<DateTime<Utc>>) -> Result<(), BillingError> {
        let now = Utc::now();
        sqlx::query(
            "UPDATE billing.payments SET status = $2, paid_at = $3, updated_at = $4 WHERE order_id = $1",
        )
        .bind(order_id)
        .bind(status)
        .bind(paid_at)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
