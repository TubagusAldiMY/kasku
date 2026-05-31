use std::sync::Arc;
use chrono::Utc;
use serde_json::json;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::auth::domain::{entity::EmailVerification, error::AuthError, repository::{EmailVerificationRepository, UserRepository}};
use crate::modules::auth::usecase::helpers::{generate_secure_token, mask_email};

const EMAIL_VERIFICATION_TTL_HOURS: i64 = 24;

pub struct ResendVerificationUseCase {
    pool: PgPool,
    user_repo: Arc<dyn UserRepository>,
    ev_repo: Arc<dyn EmailVerificationRepository>,
}

impl ResendVerificationUseCase {
    pub fn new(pool: PgPool, user_repo: Arc<dyn UserRepository>, ev_repo: Arc<dyn EmailVerificationRepository>) -> Self {
        Self { pool, user_repo, ev_repo }
    }

    pub async fn execute(&self, email: &str) -> Result<(), AuthError> {
        let user = self.user_repo.find_by_email(email).await?
            .ok_or(AuthError::UserNotFound)?;

        if user.email_verified {
            return Ok(()); // silent success — don't reveal verification status
        }

        let (raw_token, token_hash) = generate_secure_token().map_err(|e| AuthError::Internal(e))?;
        let now = Utc::now();

        let ev = EmailVerification {
            id: Uuid::new_v4(),
            user_id: user.id,
            token_hash,
            expires_at: now + chrono::Duration::hours(EMAIL_VERIFICATION_TTL_HOURS),
            verified_at: None,
            created_at: now,
        };

        let mut tx = self.pool.begin().await.map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        // Delete old verifications for this user
        sqlx::query("DELETE FROM auth.email_verifications WHERE user_id = $1")
            .bind(user.id)
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

        let payload = json!({
            "user_id": user.id.to_string(),
            "email": email,
            "verification_token": raw_token,
        });

        sqlx::query(
            "INSERT INTO auth.outbox_events (id, event_type, routing_key, payload, created_at) VALUES ($1, $2, $3, $4::jsonb, $5)",
        )
        .bind(Uuid::new_v4())
        .bind("user.email_verification_resent")
        .bind("user.email_verification_resent")
        .bind(serde_json::to_string(&payload).unwrap())
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        tx.commit().await.map_err(|e| AuthError::Internal(anyhow::anyhow!(e)))?;

        tracing::info!(email = mask_email(email), "verification email resent");
        Ok(())
    }
}
