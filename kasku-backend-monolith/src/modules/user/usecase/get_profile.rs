use std::sync::Arc;
use uuid::Uuid;

use crate::modules::user::domain::{entity::UserProfile, error::UserError, repository::UserProfileRepository};

pub struct GetProfileUseCase {
    profile_repo: Arc<dyn UserProfileRepository>,
}

impl GetProfileUseCase {
    pub fn new(profile_repo: Arc<dyn UserProfileRepository>) -> Self {
        Self { profile_repo }
    }

    pub async fn execute(&self, user_id: Uuid) -> Result<UserProfile, UserError> {
        self.profile_repo.find_by_user_id(user_id).await?
            .ok_or(UserError::ProfileNotFound)
    }
}
