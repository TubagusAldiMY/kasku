use chrono::{DateTime, Utc};
use uuid::Uuid;

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct NotificationPreference {
    pub user_id: Uuid,
    pub email_enabled: bool,
    pub payment_alerts: bool,
    pub subscription_alerts: bool,
    pub security_alerts: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}
