use async_trait::async_trait;
use chrono::Utc;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::user::domain::{entity::UserProfile, error::UserError, repository::UserProfileRepository};

pub struct PostgresUserProfileRepository {
    pool: PgPool,
}

impl PostgresUserProfileRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl UserProfileRepository for PostgresUserProfileRepository {
    async fn find_by_user_id(&self, user_id: Uuid) -> Result<Option<UserProfile>, UserError> {
        let row = sqlx::query_as::<_, UserProfile>(
            "SELECT user_id, email, username, display_name, created_at, updated_at FROM user_mgmt.user_profiles WHERE user_id = $1",
        )
        .bind(user_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    async fn upsert(&self, user_id: Uuid, email: &str, username: &str) -> Result<(), UserError> {
        let now = Utc::now();
        sqlx::query(
            r#"INSERT INTO user_mgmt.user_profiles (user_id, email, username, created_at, updated_at)
               VALUES ($1, $2, $3, $4, $4)
               ON CONFLICT (user_id) DO UPDATE SET email = EXCLUDED.email, updated_at = now()"#,
        )
        .bind(user_id)
        .bind(email)
        .bind(username)
        .bind(now)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
