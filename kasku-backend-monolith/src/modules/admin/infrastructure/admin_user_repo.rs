use async_trait::async_trait;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::admin::domain::entity::AdminUser;
use crate::modules::admin::domain::error::AdminError;
use crate::modules::admin::domain::repository::AdminUserRepository;

pub struct PostgresAdminUserRepository {
    pool: PgPool,
}

impl PostgresAdminUserRepository {
    pub fn new(pool: PgPool) -> Self { Self { pool } }
}

#[async_trait]
impl AdminUserRepository for PostgresAdminUserRepository {
    async fn find_by_username(&self, username: &str) -> Result<Option<AdminUser>, AdminError> {
        sqlx::query_as::<_, AdminUser>(
            "SELECT id, username, password_hash, role, is_active, last_login_at, created_at, updated_at
             FROM admin_panel.admin_users WHERE username = $1"
        )
        .bind(username)
        .fetch_optional(&self.pool)
        .await
        .map_err(AdminError::from)
    }

    async fn find_by_id(&self, id: Uuid) -> Result<Option<AdminUser>, AdminError> {
        sqlx::query_as::<_, AdminUser>(
            "SELECT id, username, password_hash, role, is_active, last_login_at, created_at, updated_at
             FROM admin_panel.admin_users WHERE id = $1"
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
        .map_err(AdminError::from)
    }

    async fn create(&self, user: &AdminUser) -> Result<(), AdminError> {
        sqlx::query(
            "INSERT INTO admin_panel.admin_users
             (id, username, password_hash, role, is_active, created_at, updated_at)
             VALUES ($1, $2, $3, $4, $5, $6, $7)"
        )
        .bind(user.id)
        .bind(&user.username)
        .bind(&user.password_hash)
        .bind(user.role.to_string())
        .bind(user.is_active)
        .bind(user.created_at)
        .bind(user.updated_at)
        .execute(&self.pool)
        .await
        .map(|_| ())
        .map_err(AdminError::from)
    }

    async fn update_last_login(&self, id: Uuid) -> Result<(), AdminError> {
        sqlx::query(
            "UPDATE admin_panel.admin_users SET last_login_at = NOW() WHERE id = $1"
        )
        .bind(id)
        .execute(&self.pool)
        .await
        .map(|_| ())
        .map_err(AdminError::from)
    }

    async fn count(&self) -> Result<i64, AdminError> {
        let row = sqlx::query_scalar::<_, i64>(
            "SELECT COUNT(*) FROM admin_panel.admin_users"
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AdminError::from)?;
        Ok(row)
    }
}
