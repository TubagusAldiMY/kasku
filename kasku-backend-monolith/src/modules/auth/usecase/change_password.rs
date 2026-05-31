use std::sync::Arc;
use uuid::Uuid;

use crate::modules::auth::domain::{error::AuthError, repository::UserRepository};
use crate::modules::auth::usecase::helpers::{hash_password, verify_password, Argon2Config};

pub struct ChangePasswordUseCase {
    user_repo: Arc<dyn UserRepository>,
    argon2_cfg: Argon2Config,
}

impl ChangePasswordUseCase {
    pub fn new(user_repo: Arc<dyn UserRepository>, argon2_cfg: Argon2Config) -> Self {
        Self { user_repo, argon2_cfg }
    }

    pub async fn execute(&self, user_id: Uuid, current_password: &str, new_password: &str) -> Result<(), AuthError> {
        validate_password_strength(new_password)?;

        let user = self.user_repo.find_by_id(user_id).await?
            .ok_or(AuthError::UserNotFound)?;

        let valid = verify_password(current_password, &user.password_hash)
            .map_err(|e| AuthError::Internal(e))?;

        if !valid {
            return Err(AuthError::InvalidCredentials);
        }

        let new_hash = hash_password(new_password, &self.argon2_cfg)
            .map_err(|e| AuthError::Internal(e))?;

        self.user_repo.update_password(user_id, &new_hash).await?;

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
