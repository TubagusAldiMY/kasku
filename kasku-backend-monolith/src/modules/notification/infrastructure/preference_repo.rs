use async_trait::async_trait;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::notification::domain::{
    entity::NotificationPreference,
    error::NotificationError,
    repository::PreferenceRepository,
};

pub struct PostgresPreferenceRepository {
    pool: PgPool,
}

impl PostgresPreferenceRepository {
    pub fn new(pool: PgPool) -> Self { Self { pool } }
}

#[async_trait]
impl PreferenceRepository for PostgresPreferenceRepository {
    async fn find_by_user_id(&self, user_id: Uuid) -> Result<Option<NotificationPreference>, NotificationError> {
        let row = sqlx::query_as::<_, NotificationPreference>(
            "SELECT user_id, email_enabled, payment_alerts, subscription_alerts, security_alerts, created_at, updated_at
             FROM notification.notification_preferences WHERE user_id = $1"
        )
        .bind(user_id)
        .fetch_optional(&self.pool)
        .await?;
        Ok(row)
    }

    async fn upsert(&self, pref: &NotificationPreference) -> Result<(), NotificationError> {
        sqlx::query(
            "INSERT INTO notification.notification_preferences
             (user_id, email_enabled, payment_alerts, subscription_alerts, security_alerts, created_at, updated_at)
             VALUES ($1,$2,$3,$4,$5,now(),now())
             ON CONFLICT (user_id) DO UPDATE SET
               email_enabled = EXCLUDED.email_enabled,
               payment_alerts = EXCLUDED.payment_alerts,
               subscription_alerts = EXCLUDED.subscription_alerts,
               security_alerts = EXCLUDED.security_alerts,
               updated_at = now()"
        )
        .bind(pref.user_id)
        .bind(pref.email_enabled)
        .bind(pref.payment_alerts)
        .bind(pref.subscription_alerts)
        .bind(pref.security_alerts)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
