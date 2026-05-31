use async_trait::async_trait;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::admin::domain::entity::{DashboardStats, UserSummary};
use crate::modules::admin::domain::error::AdminError;
use crate::modules::admin::domain::repository::AdminUserReadRepository;

pub struct PostgresAdminUserReadRepository {
    pool: PgPool,
}

impl PostgresAdminUserReadRepository {
    pub fn new(pool: PgPool) -> Self { Self { pool } }
}

#[async_trait]
impl AdminUserReadRepository for PostgresAdminUserReadRepository {
    async fn list_users(
        &self, page: i64, per_page: i64, search: Option<&str>,
    ) -> Result<(Vec<UserSummary>, i64), AdminError> {
        let offset = (page - 1) * per_page;
        let (users, total) = if let Some(q) = search {
            let pattern = format!("%{}%", q.to_lowercase());
            let users = sqlx::query_as::<_, UserSummary>(
                "SELECT id, email, username, is_active, email_verified, created_at, last_login_at
                 FROM auth.users
                 WHERE LOWER(email) LIKE $1 OR LOWER(username) LIKE $1
                 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
            )
            .bind(&pattern)
            .bind(per_page)
            .bind(offset)
            .fetch_all(&self.pool)
            .await
            .map_err(AdminError::from)?;

            let total = sqlx::query_scalar::<_, i64>(
                "SELECT COUNT(*) FROM auth.users WHERE LOWER(email) LIKE $1 OR LOWER(username) LIKE $1"
            )
            .bind(&pattern)
            .fetch_one(&self.pool)
            .await
            .map_err(AdminError::from)?;

            (users, total)
        } else {
            let users = sqlx::query_as::<_, UserSummary>(
                "SELECT id, email, username, is_active, email_verified, created_at, last_login_at
                 FROM auth.users ORDER BY created_at DESC LIMIT $1 OFFSET $2"
            )
            .bind(per_page)
            .bind(offset)
            .fetch_all(&self.pool)
            .await
            .map_err(AdminError::from)?;

            let total = sqlx::query_scalar::<_, i64>("SELECT COUNT(*) FROM auth.users")
                .fetch_one(&self.pool)
                .await
                .map_err(AdminError::from)?;

            (users, total)
        };
        Ok((users, total))
    }

    async fn find_user_by_id(&self, id: Uuid) -> Result<Option<UserSummary>, AdminError> {
        sqlx::query_as::<_, UserSummary>(
            "SELECT id, email, username, is_active, email_verified, created_at, last_login_at
             FROM auth.users WHERE id = $1"
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
        .map_err(AdminError::from)
    }

    async fn suspend_user(&self, id: Uuid) -> Result<(), AdminError> {
        sqlx::query("UPDATE auth.users SET is_active = false, updated_at = NOW() WHERE id = $1")
            .bind(id)
            .execute(&self.pool)
            .await
            .map(|_| ())
            .map_err(AdminError::from)
    }

    async fn activate_user(&self, id: Uuid) -> Result<(), AdminError> {
        sqlx::query("UPDATE auth.users SET is_active = true, updated_at = NOW() WHERE id = $1")
            .bind(id)
            .execute(&self.pool)
            .await
            .map(|_| ())
            .map_err(AdminError::from)
    }

    async fn dashboard_stats(&self) -> Result<DashboardStats, AdminError> {
        let total_users = sqlx::query_scalar::<_, i64>("SELECT COUNT(*) FROM auth.users")
            .fetch_one(&self.pool)
            .await
            .map_err(AdminError::from)?;

        let total_active_users = sqlx::query_scalar::<_, i64>(
            "SELECT COUNT(*) FROM auth.users WHERE is_active = true"
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AdminError::from)?;

        let new_users_last7_days = sqlx::query_scalar::<_, i64>(
            "SELECT COUNT(*) FROM auth.users WHERE created_at >= NOW() - INTERVAL '7 days'"
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AdminError::from)?;

        Ok(DashboardStats { total_users, total_active_users, new_users_last7_days })
    }
}
