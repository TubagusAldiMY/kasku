use std::sync::Arc;
use chrono::Utc;

use crate::modules::auth::domain::{error::AuthError, repository::{PasswordResetRepository, UserRepository}};
use crate::modules::auth::usecase::helpers::{hash_password, sha256_hex, Argon2Config};

pub struct ResetPasswordUseCase {
    user_repo: Arc<dyn UserRepository>,
    reset_repo: Arc<dyn PasswordResetRepository>,
    argon2_cfg: Argon2Config,
}

impl ResetPasswordUseCase {
    pub fn new(
        user_repo: Arc<dyn UserRepository>,
        reset_repo: Arc<dyn PasswordResetRepository>,
        argon2_cfg: Argon2Config,
    ) -> Self {
        Self { user_repo, reset_repo, argon2_cfg }
    }

    pub async fn execute(&self, raw_token: &str, new_password: &str) -> Result<(), AuthError> {
        validate_password_strength(new_password)?;

        let hash = sha256_hex(raw_token);
        let token = self.reset_repo.find_by_hash(&hash).await?
            .ok_or(AuthError::InvalidToken)?;

        if token.is_used() {
            return Err(AuthError::TokenAlreadyUsed);
        }
        if token.is_expired(Utc::now()) {
            return Err(AuthError::InvalidToken);
        }

        let new_hash = hash_password(new_password, &self.argon2_cfg)
            .map_err(|e| AuthError::Internal(e))?;

        self.user_repo.update_password(token.user_id, &new_hash).await?;
        self.reset_repo.mark_used(token.id).await?;

        Ok(())
    }
}

fn validate_password_strength(password: &str) -> Result<(), AuthError> {
    if password.len() < 8 {
        return Err(AuthError::PasswordTooShort);
    }
    let has_upper = password.chars().any(|c| c.is_uppercase());
    let has_lower = password.chars().any(|c| c.is_lowercase());
    let has_digit = password.chars().any(|c| c.is_ascii_digit());
    if !has_upper || !has_lower || !has_digit {
        return Err(AuthError::PasswordTooWeak);
    }
    Ok(())
}
