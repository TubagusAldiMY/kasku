use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::auth::domain::{
    entity::RefreshToken,
    error::AuthError,
    repository::RefreshTokenRepository,
};

pub struct PostgresRefreshTokenRepository {
    pool: PgPool,
}

impl PostgresRefreshTokenRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl RefreshTokenRepository for PostgresRefreshTokenRepository {
    async fn create(&self, token: &RefreshToken) -> Result<(), AuthError> {
        sqlx::query(
            r#"INSERT INTO auth.refresh_tokens
               (id, user_id, token_hash, user_agent, ip_address, expires_at, is_revoked, created_at)
               VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"#,
        )
        .bind(token.id)
        .bind(token.user_id)
        .bind(&token.token_hash)
        .bind(&token.user_agent)
        .bind(&token.ip_address)
        .bind(token.expires_at)
        .bind(token.is_revoked)
        .bind(token.created_at)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn find_by_hash(&self, hash: &str) -> Result<Option<RefreshToken>, AuthError> {
        let row = sqlx::query_as::<_, RefreshToken>(
            r#"SELECT id, user_id, token_hash, user_agent, ip_address,
               expires_at, is_revoked, revoked_at, created_at
               FROM auth.refresh_tokens WHERE token_hash = $1 LIMIT 1"#,
        )
        .bind(hash)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn revoke(&self, token_id: Uuid) -> Result<(), AuthError> {
        let now = Utc::now();
        sqlx::query(
            "UPDATE auth.refresh_tokens SET is_revoked = true, revoked_at = $2 WHERE id = $1",
        )
        .bind(token_id)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn revoke_all_for_user(&self, user_id: Uuid) -> Result<(), AuthError> {
        let now = Utc::now();
        sqlx::query(
            "UPDATE auth.refresh_tokens SET is_revoked = true, revoked_at = $2 WHERE user_id = $1 AND is_revoked = false",
        )
        .bind(user_id)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn delete_expired(&self, before: DateTime<Utc>) -> Result<u64, AuthError> {
        let result = sqlx::query(
            "DELETE FROM auth.refresh_tokens WHERE expires_at < $1",
        )
        .bind(before)
        .execute(&self.pool)
        .await?;
        Ok(result.rows_affected())
    }
}
