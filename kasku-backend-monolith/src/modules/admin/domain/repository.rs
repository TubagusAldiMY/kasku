use async_trait::async_trait;
use uuid::Uuid;

use super::entity::{AdminUser, AuditLogEntry, DashboardStats, UserSummary};
use super::error::AdminError;

#[async_trait]
pub trait AdminUserRepository: Send + Sync {
    async fn find_by_username(&self, username: &str) -> Result<Option<AdminUser>, AdminError>;
    async fn find_by_id(&self, id: Uuid) -> Result<Option<AdminUser>, AdminError>;
    async fn create(&self, user: &AdminUser) -> Result<(), AdminError>;
    async fn update_last_login(&self, id: Uuid) -> Result<(), AdminError>;
    async fn count(&self) -> Result<i64, AdminError>;
}

#[async_trait]
pub trait AdminUserReadRepository: Send + Sync {
    async fn list_users(
        &self, page: i64, per_page: i64, search: Option<&str>,
    ) -> Result<(Vec<UserSummary>, i64), AdminError>;
    async fn find_user_by_id(&self, id: Uuid) -> Result<Option<UserSummary>, AdminError>;
    async fn suspend_user(&self, id: Uuid) -> Result<(), AdminError>;
    async fn activate_user(&self, id: Uuid) -> Result<(), AdminError>;
    async fn dashboard_stats(&self) -> Result<DashboardStats, AdminError>;
}

#[async_trait]
pub trait AuditLogRepository: Send + Sync {
    async fn create(&self, entry: &AuditLogEntry) -> Result<(), AdminError>;
    async fn list(
        &self, page: i64, per_page: i64, admin_id: Option<Uuid>,
    ) -> Result<(Vec<AuditLogEntry>, i64), AdminError>;
}
