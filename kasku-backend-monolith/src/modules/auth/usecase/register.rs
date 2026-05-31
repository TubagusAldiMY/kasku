use std::sync::Arc;
use chrono::Utc;
use serde_json::json;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::auth::domain::{entity::{EmailVerification, User}, error::AuthError, repository::{EmailVerificationRepository, UserRepository}};
use crate::modules::auth::usecase::helpers::{generate_secure_token, hash_password, mask_email, Argon2Config};
use crate::modules::user::usecase::ProvisionTenantUseCase;

const EMAIL_VERIFICATION_TTL_HOURS: i64 = 24;

pub struct RegisterInput {
    pub email: String,
    pub username: String,
    pub password: String,
}

pub struct RegisterOutput {
    pub user_id: Uuid,
    pub email: String,
    pub username: String,
}

pub struct RegisterUseCase {
    pool: PgPool,
    user_repo: Arc<dyn UserRepository>,
    email_verification_repo: Arc<dyn EmailVerificationRepository>,
    provision_uc: Arc<ProvisionTenantUseCase>,
    argon2_cfg: Argon2Config,
}

impl RegisterUseCase {
    pub fn new(
        pool: PgPool,
        user_repo: Arc<dyn UserRepository>,
        email_verification_repo: Arc<dyn EmailVerificationRepository>,
        provision_uc: Arc<ProvisionTenantUseCase>,
        argon2_cfg: Argon2Config,
    ) -> Self {
        Self { pool, user_repo, email_verification_repo, provision_uc, argon2_cfg }
    }

    pub async fn execute(&self, input: RegisterInput) -> Result<RegisterOutput, AuthError> {
        validate_register_input(&input)?;

        if self.user_repo.exists_by_email(&input.email).await? {
            return Err(AuthError::EmailAlreadyExists);
        }
        if self.user_repo.exists_by_username(&input.username).await? {
            return Err(AuthError::UsernameAlreadyExists);
        }

        let password_hash = hash_password(&input.password, &self.argon2_cfg)
            .map_err(|e| AuthError::Internal(e))?;

        let (raw_token, token_hash) = generate_secure_token()
            .map_err(|e| AuthError::Internal(e))?;

        let now = Utc::now();
        let user_id = Uuid::new_v4();

        let new_user = User {
            id: user_id,
            email: input.email.clone(),
            username: input.username.clone(),
            password_hash,
            is_active: false,
            email_verified: false,
            failed_login_count: 0,
            locked_until: None,
            last_login_at: None,
            created_at: now,
            updated_at: now,
        };

        let ev = EmailVerification {
            id: Uuid::new_v4(),
            user_id,
            token_hash,
            expires_at: now + chrono::Duration::hours(EMAIL_VERIFICATION_TTL_HOURS),
            verified_at: None,
            created_at: now,
        };

        // Atomic transaction: INSERT user + INSERT email_verification + INSERT outbox event
        let mut tx = self.pool.begin().await.map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        sqlx::query(
            r#"INSERT INTO auth.users
               (id, email, username, password_hash, is_active, email_verified, failed_login_count, created_at, updated_at)
               VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"#,
        )
        .bind(new_user.id)
        .bind(&new_user.email)
        .bind(&new_user.username)
        .bind(&new_user.password_hash)
        .bind(new_user.is_active)
        .bind(new_user.email_verified)
        .bind(new_user.failed_login_count)
        .bind(new_user.created_at)
        .bind(new_user.updated_at)
        .execute(&mut *tx)
        .await
        .map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        sqlx::query(
            "INSERT INTO auth.email_verifications (id, user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)",
        )
        .bind(ev.id)
        .bind(ev.user_id)
        .bind(&ev.token_hash)
        .bind(ev.expires_at)
        .bind(ev.created_at)
        .execute(&mut *tx)
        .await
        .map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        let event_payload = json!({
            "user_id": user_id.to_string(),
            "email": input.email,
            "username": input.username,
            "verification_token": raw_token,
        });

        sqlx::query(
            r#"INSERT INTO auth.outbox_events (id, event_type, routing_key, payload, created_at)
               VALUES ($1, $2, $3, $4::jsonb, $5)"#,
        )
        .bind(Uuid::new_v4())
        .bind("user.registered")
        .bind("user.registered")
        .bind(serde_json::to_string(&event_payload).unwrap())
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        tx.commit().await.map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        // In-process provisioning after commit — replaces old RabbitMQ-triggered flow
        if let Err(e) = self.provision_uc.execute(user_id, &input.email, &input.username).await {
            tracing::error!(
                user_id = %user_id,
                email = mask_email(&input.email),
                error = %e,
                "provision tenant failed after registration"
            );
            // Don't fail registration — user can still verify email; provisioning retried on login
        }

        Ok(RegisterOutput {
            user_id,
            email: input.email,
            username: input.username,
        })
    }
}

fn validate_register_input(input: &RegisterInput) -> Result<(), AuthError> {
    // Email format
    if !is_valid_email(&input.email) {
        return Err(AuthError::Validation("format email tidak valid".into()));
    }

    // Username: 3–30 chars, alphanumeric + underscore
    let ulen = input.username.len();
    if ulen < 3 || ulen > 30 {
        return Err(AuthError::Validation("username harus 3-30 karakter".into()));
    }
    if !input.username.chars().all(|c| c.is_alphanumeric() || c == '_') {
        return Err(AuthError::Validation("username hanya boleh berisi huruf, angka, dan underscore".into()));
    }

    validate_password(&input.password)
}

fn validate_password(password: &str) -> Result<(), AuthError> {
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

fn is_valid_email(email: &str) -> bool {
    let parts: Vec<&str> = email.splitn(2, '@').collect();
    if parts.len() != 2 || parts[0].is_empty() || parts[1].is_empty() {
        return false;
    }
    parts[1].contains('.')
}
