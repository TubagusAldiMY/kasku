use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::billing::domain::{
    entity::Subscription,
    error::BillingError,
    repository::SubscriptionRepository,
};

pub struct PostgresSubscriptionRepository {
    pool: PgPool,
}

impl PostgresSubscriptionRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[derive(sqlx::FromRow)]
struct SubscriptionWithPlanName {
    pub id: Uuid,
    pub user_id: Uuid,
    pub plan_id: Uuid,
    pub status: String,
    pub current_period_start: DateTime<Utc>,
    pub current_period_end: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub plan_name: String,
}

#[async_trait]
impl SubscriptionRepository for PostgresSubscriptionRepository {
    async fn find_by_user_id(&self, user_id: Uuid) -> Result<Option<Subscription>, BillingError> {
        let row = sqlx::query_as::<_, Subscription>(
            r#"SELECT id, user_id, plan_id, status, current_period_start, current_period_end, created_at, updated_at
               FROM billing.subscriptions WHERE user_id = $1 LIMIT 1"#,
        )
        .bind(user_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn create(&self, sub: &Subscription) -> Result<(), BillingError> {
        sqlx::query(
            r#"INSERT INTO billing.subscriptions
               (id, user_id, plan_id, status, current_period_start, current_period_end, created_at, updated_at)
               VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"#,
        )
        .bind(sub.id)
        .bind(sub.user_id)
        .bind(sub.plan_id)
        .bind(&sub.status)
        .bind(sub.current_period_start)
        .bind(sub.current_period_end)
        .bind(sub.created_at)
        .bind(sub.updated_at)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn update_plan(
        &self,
        user_id: Uuid,
        plan_id: Uuid,
        period_end: Option<DateTime<Utc>>,
        status: &str,
    ) -> Result<(), BillingError> {
        let now = Utc::now();
        sqlx::query(
            r#"UPDATE billing.subscriptions
               SET plan_id = $2, status = $3, current_period_end = $4,
                   current_period_start = $5, updated_at = $5
               WHERE user_id = $1"#,
        )
        .bind(user_id)
        .bind(plan_id)
        .bind(status)
        .bind(period_end)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn find_expiring(&self, within_hours: i64) -> Result<Vec<(Subscription, String)>, BillingError> {
        let cutoff = Utc::now() + chrono::Duration::hours(within_hours);
        let rows = sqlx::query_as::<_, SubscriptionWithPlanName>(
            r#"SELECT s.id, s.user_id, s.plan_id, s.status,
               s.current_period_start, s.current_period_end, s.created_at, s.updated_at,
               p.name as plan_name
               FROM billing.subscriptions s
               JOIN billing.subscription_plans p ON p.id = s.plan_id
               WHERE s.status = 'ACTIVE'
                 AND s.current_period_end IS NOT NULL
                 AND s.current_period_end <= $1"#,
        )
        .bind(cutoff)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| {
            (Subscription {
                id: r.id,
                user_id: r.user_id,
                plan_id: r.plan_id,
                status: r.status,
                current_period_start: r.current_period_start,
                current_period_end: r.current_period_end,
                created_at: r.created_at,
                updated_at: r.updated_at,
            }, r.plan_name)
        }).collect())
    }

    async fn expire_overdue(&self) -> Result<u64, BillingError> {
        let now = Utc::now();
        let result = sqlx::query(
            r#"UPDATE billing.subscriptions
               SET status = 'EXPIRED', updated_at = $1
               WHERE status = 'ACTIVE'
                 AND current_period_end IS NOT NULL
                 AND current_period_end < $1"#,
        )
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(result.rows_affected())
    }
}
