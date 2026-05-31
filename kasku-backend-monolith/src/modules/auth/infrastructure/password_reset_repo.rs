use async_trait::async_trait;
use chrono::Utc;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::auth::domain::{
    entity::PasswordResetToken,
    error::AuthError,
    repository::PasswordResetRepository,
};

pub struct PostgresPasswordResetRepository {
    pool: PgPool,
}

impl PostgresPasswordResetRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl PasswordResetRepository for PostgresPasswordResetRepository {
    async fn create(&self, token: &PasswordResetToken) -> Result<(), AuthError> {
        sqlx::query(
            r#"INSERT INTO auth.password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
               VALUES ($1, $2, $3, $4, $5)"#,
        )
        .bind(token.id)
        .bind(token.user_id)
        .bind(&token.token_hash)
        .bind(token.expires_at)
        .bind(token.created_at)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn find_by_hash(&self, hash: &str) -> Result<Option<PasswordResetToken>, AuthError> {
        let row = sqlx::query_as::<_, PasswordResetToken>(
            "SELECT id, user_id, token_hash, expires_at, used_at, created_at
             FROM auth.password_reset_tokens WHERE token_hash = $1 LIMIT 1",
        )
        .bind(hash)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn mark_used(&self, id: Uuid) -> Result<(), AuthError> {
        let now = Utc::now();
        sqlx::query(
            "UPDATE auth.password_reset_tokens SET used_at = $2 WHERE id = $1",
        )
        .bind(id)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn delete_for_user(&self, user_id: Uuid) -> Result<(), AuthError> {
        sqlx::query(
            "DELETE FROM auth.password_reset_tokens WHERE user_id = $1",
        )
        .bind(user_id)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
