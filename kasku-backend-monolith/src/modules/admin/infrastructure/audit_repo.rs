use async_trait::async_trait;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::admin::domain::entity::AuditLogEntry;
use crate::modules::admin::domain::error::AdminError;
use crate::modules::admin::domain::repository::AuditLogRepository;

pub struct PostgresAuditLogRepository {
    pool: PgPool,
}

impl PostgresAuditLogRepository {
    pub fn new(pool: PgPool) -> Self { Self { pool } }
}

#[async_trait]
impl AuditLogRepository for PostgresAuditLogRepository {
    async fn create(&self, entry: &AuditLogEntry) -> Result<(), AdminError> {
        sqlx::query(
            "INSERT INTO admin_panel.admin_audit_log
             (id, admin_id, action, target_user_id, target_entity, metadata, success, created_at)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
        )
        .bind(entry.id)
        .bind(entry.admin_id)
        .bind(entry.action.to_string())
        .bind(entry.target_user_id)
        .bind(&entry.target_entity)
        .bind(&entry.metadata)
        .bind(entry.success)
        .bind(entry.created_at)
        .execute(&self.pool)
        .await
        .map(|_| ())
        .map_err(AdminError::from)
    }

    async fn list(
        &self, page: i64, per_page: i64, admin_id: Option<Uuid>,
    ) -> Result<(Vec<AuditLogEntry>, i64), AdminError> {
        let offset = (page - 1) * per_page;
        let (entries, total) = if let Some(aid) = admin_id {
            let entries = sqlx::query_as::<_, AuditLogEntry>(
                "SELECT id, admin_id, action, target_user_id, target_entity, metadata, success, created_at
                 FROM admin_panel.admin_audit_log
                 WHERE admin_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
            )
            .bind(aid).bind(per_page).bind(offset)
            .fetch_all(&self.pool).await.map_err(AdminError::from)?;

            let total = sqlx::query_scalar::<_, i64>(
                "SELECT COUNT(*) FROM admin_panel.admin_audit_log WHERE admin_id = $1"
            )
            .bind(aid)
            .fetch_one(&self.pool).await.map_err(AdminError::from)?;

            (entries, total)
        } else {
            let entries = sqlx::query_as::<_, AuditLogEntry>(
                "SELECT id, admin_id, action, target_user_id, target_entity, metadata, success, created_at
                 FROM admin_panel.admin_audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2"
            )
            .bind(per_page).bind(offset)
            .fetch_all(&self.pool).await.map_err(AdminError::from)?;

            let total = sqlx::query_scalar::<_, i64>(
                "SELECT COUNT(*) FROM admin_panel.admin_audit_log"
            )
            .fetch_one(&self.pool).await.map_err(AdminError::from)?;

            (entries, total)
        };
        Ok((entries, total))
    }
}
