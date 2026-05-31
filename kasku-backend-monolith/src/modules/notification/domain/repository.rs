use async_trait::async_trait;
use uuid::Uuid;
use super::{entity::NotificationPreference, error::NotificationError};

#[async_trait]
pub trait PreferenceRepository: Send + Sync {
    async fn find_by_user_id(&self, user_id: Uuid) -> Result<Option<NotificationPreference>, NotificationError>;
    async fn upsert(&self, pref: &NotificationPreference) -> Result<(), NotificationError>;
}
