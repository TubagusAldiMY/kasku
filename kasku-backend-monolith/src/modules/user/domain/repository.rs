use async_trait::async_trait;
use uuid::Uuid;
use super::{entity::UserProfile, error::UserError};

#[async_trait]
pub trait UserProfileRepository: Send + Sync {
    async fn find_by_user_id(&self, user_id: Uuid) -> Result<Option<UserProfile>, UserError>;
    async fn upsert(&self, user_id: Uuid, email: &str, username: &str) -> Result<(), UserError>;
}
