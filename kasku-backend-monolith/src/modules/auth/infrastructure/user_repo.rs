use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::auth::domain::{
    entity::User,
    error::AuthError,
    repository::UserRepository,
};

pub struct PostgresUserRepository {
    pool: PgPool,
}

impl PostgresUserRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl UserRepository for PostgresUserRepository {
    async fn find_by_email(&self, email: &str) -> Result<Option<User>, AuthError> {
        let row = sqlx::query_as::<_, User>(
            r#"SELECT id, email, username, password_hash, is_active, email_verified,
               failed_login_count, locked_until, last_login_at, created_at, updated_at
               FROM auth.users WHERE LOWER(email) = LOWER($1) LIMIT 1"#,
        )
        .bind(email)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn find_by_id(&self, id: Uuid) -> Result<Option<User>, AuthError> {
        let row = sqlx::query_as::<_, User>(
            r#"SELECT id, email, username, password_hash, is_active, email_verified,
               failed_login_count, locked_until, last_login_at, created_at, updated_at
               FROM auth.users WHERE id = $1 LIMIT 1"#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn exists_by_email(&self, email: &str) -> Result<bool, AuthError> {
        let row: Option<bool> = sqlx::query_scalar(
            "SELECT EXISTS(SELECT 1 FROM auth.users WHERE LOWER(email) = LOWER($1))",
        )
        .bind(email)
        .fetch_one(&self.pool)
        .await?;
        Ok(row.unwrap_or(false))
    }

    async fn exists_by_username(&self, username: &str) -> Result<bool, AuthError> {
        let row: Option<bool> = sqlx::query_scalar(
            "SELECT EXISTS(SELECT 1 FROM auth.users WHERE LOWER(username) = LOWER($1))",
        )
        .bind(username)
        .fetch_one(&self.pool)
        .await?;
        Ok(row.unwrap_or(false))
    }

    async fn create(&self, user: &User) -> Result<(), AuthError> {
        sqlx::query(
            r#"INSERT INTO auth.users
               (id, email, username, password_hash, is_active, email_verified,
                failed_login_count, created_at, updated_at)
               VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"#,
        )
        .bind(user.id)
        .bind(&user.email)
        .bind(&user.username)
        .bind(&user.password_hash)
        .bind(user.is_active)
        .bind(user.email_verified)
        .bind(user.failed_login_count)
        .bind(user.created_at)
        .bind(user.updated_at)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn update_login_success(&self, user_id: Uuid) -> Result<(), AuthError> {
        let now = Utc::now();
        sqlx::query(
            r#"UPDATE auth.users
               SET failed_login_count = 0, last_login_at = $2, locked_until = NULL, updated_at = $3
               WHERE id = $1"#,
        )
        .bind(user_id)
        .bind(now)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn increment_failed_login_and_lock(
        &self,
        user_id: Uuid,
        max_attempts: i16,
        lockout_secs: i64,
    ) -> Result<(), AuthError> {
        let now = Utc::now();
        let lock_until = now + chrono::Duration::seconds(lockout_secs);
        sqlx::query(
            r#"UPDATE auth.users
               SET
                 failed_login_count = LEAST(failed_login_count + 1, $2),
                 locked_until = CASE
                   WHEN failed_login_count + 1 >= $2 THEN $3
                   ELSE locked_until
                 END,
                 updated_at = $4
               WHERE id = $1"#,
        )
        .bind(user_id)
        .bind(max_attempts)
        .bind(lock_until)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn verify_email(&self, user_id: Uuid) -> Result<(), AuthError> {
        let now = Utc::now();
        sqlx::query(
            "UPDATE auth.users SET is_active = true, email_verified = true, updated_at = $2 WHERE id = $1",
        )
        .bind(user_id)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn update_password(&self, user_id: Uuid, new_hash: &str) -> Result<(), AuthError> {
        let now = Utc::now();
        sqlx::query(
            "UPDATE auth.users SET password_hash = $2, updated_at = $3 WHERE id = $1",
        )
        .bind(user_id)
        .bind(new_hash)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    async fn set_active(&self, user_id: Uuid, active: bool) -> Result<(), AuthError> {
        let now = Utc::now();
        sqlx::query(
            "UPDATE auth.users SET is_active = $2, updated_at = $3 WHERE id = $1",
        )
        .bind(user_id)
        .bind(active)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
