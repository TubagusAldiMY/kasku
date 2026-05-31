use async_trait::async_trait;
use uuid::Uuid;
use chrono::DateTime;
use chrono::Utc;

use super::entity::{EmailVerification, PasswordResetToken, RefreshToken, User};
use super::error::AuthError;

#[async_trait]
pub trait UserRepository: Send + Sync {
    async fn find_by_email(&self, email: &str) -> Result<Option<User>, AuthError>;
    async fn find_by_id(&self, id: Uuid) -> Result<Option<User>, AuthError>;
    async fn exists_by_email(&self, email: &str) -> Result<bool, AuthError>;
    async fn exists_by_username(&self, username: &str) -> Result<bool, AuthError>;
    async fn create(&self, user: &User) -> Result<(), AuthError>;
    async fn update_login_success(&self, user_id: Uuid) -> Result<(), AuthError>;
    async fn increment_failed_login_and_lock(
        &self,
        user_id: Uuid,
        max_attempts: i16,
        lockout_secs: i64,
    ) -> Result<(), AuthError>;
    async fn verify_email(&self, user_id: Uuid) -> Result<(), AuthError>;
    async fn update_password(&self, user_id: Uuid, new_hash: &str) -> Result<(), AuthError>;
    async fn set_active(&self, user_id: Uuid, active: bool) -> Result<(), AuthError>;
}

#[async_trait]
pub trait RefreshTokenRepository: Send + Sync {
    async fn create(&self, token: &RefreshToken) -> Result<(), AuthError>;
    async fn find_by_hash(&self, hash: &str) -> Result<Option<RefreshToken>, AuthError>;
    async fn revoke(&self, token_id: Uuid) -> Result<(), AuthError>;
    async fn revoke_all_for_user(&self, user_id: Uuid) -> Result<(), AuthError>;
    async fn delete_expired(&self, before: DateTime<Utc>) -> Result<u64, AuthError>;
}

#[async_trait]
pub trait EmailVerificationRepository: Send + Sync {
    async fn create(&self, ev: &EmailVerification) -> Result<(), AuthError>;
    async fn find_by_hash(&self, hash: &str) -> Result<Option<EmailVerification>, AuthError>;
    async fn mark_verified(&self, id: Uuid) -> Result<(), AuthError>;
    async fn delete_for_user(&self, user_id: Uuid) -> Result<(), AuthError>;
}

#[async_trait]
pub trait PasswordResetRepository: Send + Sync {
    async fn create(&self, token: &PasswordResetToken) -> Result<(), AuthError>;
    async fn find_by_hash(&self, hash: &str) -> Result<Option<PasswordResetToken>, AuthError>;
    async fn mark_used(&self, id: Uuid) -> Result<(), AuthError>;
    async fn delete_for_user(&self, user_id: Uuid) -> Result<(), AuthError>;
}
