use chrono::{DateTime, Utc};
use uuid::Uuid;

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct User {
    pub id: Uuid,
    pub email: String,
    pub username: String,
    pub password_hash: String,
    pub is_active: bool,
    pub email_verified: bool,
    pub failed_login_count: i16,
    pub locked_until: Option<DateTime<Utc>>,
    pub last_login_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl User {
    pub fn is_account_locked(&self, now: DateTime<Utc>) -> bool {
        self.locked_until.map_or(false, |t| t > now)
    }
}

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct RefreshToken {
    pub id: Uuid,
    pub user_id: Uuid,
    pub token_hash: String,
    pub user_agent: Option<String>,
    pub ip_address: Option<String>,
    pub expires_at: DateTime<Utc>,
    pub is_revoked: bool,
    pub revoked_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

impl RefreshToken {
    pub fn is_expired(&self, now: DateTime<Utc>) -> bool {
        self.expires_at < now
    }
}

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct EmailVerification {
    pub id: Uuid,
    pub user_id: Uuid,
    pub token_hash: String,
    pub expires_at: DateTime<Utc>,
    pub verified_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

impl EmailVerification {
    pub fn is_expired(&self, now: DateTime<Utc>) -> bool {
        self.expires_at < now
    }

    pub fn is_used(&self) -> bool {
        self.verified_at.is_some()
    }
}

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct PasswordResetToken {
    pub id: Uuid,
    pub user_id: Uuid,
    pub token_hash: String,
    pub expires_at: DateTime<Utc>,
    pub used_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

impl PasswordResetToken {
    pub fn is_expired(&self, now: DateTime<Utc>) -> bool {
        self.expires_at < now
    }

    pub fn is_used(&self) -> bool {
        self.used_at.is_some()
    }
}
