use async_trait::async_trait;
use chrono::Utc;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::auth::domain::{
    entity::EmailVerification,
    error::AuthError,
    repository::EmailVerificationRepository,
};

pub struct PostgresEmailVerificationRepository {
    pool: PgPool,
}

impl PostgresEmailVerificationRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl EmailVerificationRepository for PostgresEmailVerificationRepository {
    async fn create(&self, ev: &EmailVerification) -> Result<(), AuthError> {
        sqlx::query(
            r#"INSERT INTO auth.email_verifications (id, user_id, token_hash, expires_at, created_at)
               VALUES ($1, $2, $3, $4, $5)"#,
        )
        .bind(ev.id)
        .bind(ev.user_id)
        .bind(&ev.token_hash)
        .bind(ev.expires_at)
        .bind(ev.created_at)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn find_by_hash(&self, hash: &str) -> Result<Option<EmailVerification>, AuthError> {
        let row = sqlx::query_as::<_, EmailVerification>(
            "SELECT id, user_id, token_hash, expires_at, verified_at, created_at
             FROM auth.email_verifications WHERE token_hash = $1 LIMIT 1",
        )
        .bind(hash)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn mark_verified(&self, id: Uuid) -> Result<(), AuthError> {
        let now = Utc::now();
        sqlx::query(
            "UPDATE auth.email_verifications SET verified_at = $2 WHERE id = $1",
        )
        .bind(id)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn delete_for_user(&self, user_id: Uuid) -> Result<(), AuthError> {
        sqlx::query(
            "DELETE FROM auth.email_verifications WHERE user_id = $1",
        )
        .bind(user_id)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
