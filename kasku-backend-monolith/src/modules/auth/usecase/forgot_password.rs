use std::sync::Arc;
use chrono::Utc;
use serde_json::json;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::auth::domain::{entity::PasswordResetToken, error::AuthError, repository::{PasswordResetRepository, UserRepository}};
use crate::modules::auth::usecase::helpers::{generate_secure_token, mask_email};

const RESET_TOKEN_TTL_HOURS: i64 = 1;

pub struct ForgotPasswordUseCase {
    pool: PgPool,
    user_repo: Arc<dyn UserRepository>,
    reset_repo: Arc<dyn PasswordResetRepository>,
}

impl ForgotPasswordUseCase {
    pub fn new(pool: PgPool, user_repo: Arc<dyn UserRepository>, reset_repo: Arc<dyn PasswordResetRepository>) -> Self {
        Self { pool, user_repo, reset_repo }
    }

    pub async fn execute(&self, email: &str) -> Result<(), AuthError> {
        // Silent success even if user not found — don't reveal account existence
        let user = match self.user_repo.find_by_email(email).await? {
            Some(u) => u,
            None => return Ok(()),
        };

        let (raw_token, token_hash) = generate_secure_token().map_err(|e| AuthError::Internal(e))?;
        let now = Utc::now();

        let token = PasswordResetToken {
            id: Uuid::new_v4(),
            user_id: user.id,
            token_hash,
            expires_at: now + chrono::Duration::hours(RESET_TOKEN_TTL_HOURS),
            used_at: None,
            created_at: now,
        };

        let mut tx = self.pool.begin().await.map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        // Invalidate old tokens
        sqlx::query("DELETE FROM auth.password_reset_tokens WHERE user_id = $1")
            .bind(user.id)
            .execute(&mut *tx)
            .await
            .map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        sqlx::query(
            "INSERT INTO auth.password_reset_tokens (id, user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)",
        )
        .bind(token.id)
        .bind(token.user_id)
        .bind(&token.token_hash)
        .bind(token.expires_at)
        .bind(token.created_at)
        .execute(&mut *tx)
        .await
        .map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        let payload = json!({
            "user_id": user.id.to_string(),
            "email": email,
            "reset_token": raw_token,
        });

        sqlx::query(
            "INSERT INTO auth.outbox_events (id, event_type, routing_key, payload, created_at) VALUES ($1, $2, $3, $4::jsonb, $5)",
        )
        .bind(Uuid::new_v4())
        .bind("user.password_reset_requested")
        .bind("user.password_reset_requested")
        .bind(serde_json::to_string(&payload).unwrap())
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        tx.commit().await.map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        tracing::info!(email = mask_email(email), "password reset requested");
        Ok(())
    }
}
