use std::sync::Arc;
use chrono::Utc;

use crate::modules::auth::domain::{error::AuthError, repository::{EmailVerificationRepository, UserRepository}};
use crate::modules::auth::usecase::helpers::sha256_hex;

pub struct VerifyEmailUseCase {
    user_repo: Arc<dyn UserRepository>,
    ev_repo: Arc<dyn EmailVerificationRepository>,
}

impl VerifyEmailUseCase {
    pub fn new(user_repo: Arc<dyn UserRepository>, ev_repo: Arc<dyn EmailVerificationRepository>) -> Self {
        Self { user_repo, ev_repo }
    }

    pub async fn execute(&self, raw_token: &str) -> Result<(), AuthError> {
        let hash = sha256_hex(raw_token);
        let ev = self.ev_repo.find_by_hash(&hash).await?
            .ok_or(AuthError::InvalidToken)?;

        if ev.is_used() {
            return Err(AuthError::TokenAlreadyUsed);
        }
        if ev.is_expired(Utc::now()) {
            return Err(AuthError::InvalidToken);
        }

        self.ev_repo.mark_verified(ev.id).await?;
        self.user_repo.verify_email(ev.user_id).await?;

        Ok(())
    }
}
