use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::modules::admin::domain::entity::{AdminUser, AuditLogEntry, DashboardStats, UserSummary};

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub username: String,
    pub password: String,
}

#[derive(Debug, Serialize)]
pub struct LoginResponse {
    pub token: String,
    pub admin: AdminResponse,
}

#[derive(Debug, Serialize)]
pub struct AdminResponse {
    pub id: Uuid,
    pub username: String,
    pub role: String,
    pub is_active: bool,
    pub last_login_at: Option<DateTime<Utc>>,
}

impl From<&AdminUser> for AdminResponse {
    fn from(a: &AdminUser) -> Self {
        Self {
            id: a.id,
            username: a.username.clone(),
            role: a.role.to_string(),
            is_active: a.is_active,
            last_login_at: a.last_login_at,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct UserSummaryResponse {
    pub id: Uuid,
    pub email: String,
    pub username: String,
    pub is_active: bool,
    pub email_verified: bool,
    pub created_at: DateTime<Utc>,
    pub last_login_at: Option<DateTime<Utc>>,
}

impl From<&UserSummary> for UserSummaryResponse {
    fn from(u: &UserSummary) -> Self {
        Self {
            id: u.id,
            email: u.email.clone(),
            username: u.username.clone(),
            is_active: u.is_active,
            email_verified: u.email_verified,
            created_at: u.created_at,
            last_login_at: u.last_login_at,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct PaginatedUsersResponse {
    pub users: Vec<UserSummaryResponse>,
    pub total: i64,
    pub page: i64,
    pub per_page: i64,
}

#[derive(Debug, Deserialize)]
pub struct ListUsersQuery {
    #[serde(default = "default_page")]
    pub page: i64,
    #[serde(default = "default_per_page")]
    pub per_page: i64,
    pub search: Option<String>,
}

fn default_page() -> i64 { 1 }
fn default_per_page() -> i64 { 20 }

#[derive(Debug, Deserialize)]
pub struct ListAuditQuery {
    #[serde(default = "default_page")]
    pub page: i64,
    #[serde(default = "default_per_page")]
    pub per_page: i64,
    pub admin_id: Option<Uuid>,
}

#[derive(Debug, Serialize)]
pub struct AuditLogResponse {
    pub id: Uuid,
    pub admin_id: Uuid,
    pub action: String,
    pub target_user_id: Option<Uuid>,
    pub target_entity: Option<String>,
    pub metadata: serde_json::Value,
    pub success: bool,
    pub created_at: DateTime<Utc>,
}

impl From<&AuditLogEntry> for AuditLogResponse {
    fn from(e: &AuditLogEntry) -> Self {
        Self {
            id: e.id,
            admin_id: e.admin_id,
            action: e.action.to_string(),
            target_user_id: e.target_user_id,
            target_entity: e.target_entity.clone(),
            metadata: e.metadata.clone(),
            success: e.success,
            created_at: e.created_at,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct PaginatedAuditResponse {
    pub entries: Vec<AuditLogResponse>,
    pub total: i64,
    pub page: i64,
    pub per_page: i64,
}

#[derive(Debug, Serialize)]
pub struct DashboardStatsResponse {
    pub total_users: i64,
    pub total_active_users: i64,
    pub new_users_last7_days: i64,
}

impl From<&DashboardStats> for DashboardStatsResponse {
    fn from(s: &DashboardStats) -> Self {
        Self {
            total_users: s.total_users,
            total_active_users: s.total_active_users,
            new_users_last7_days: s.new_users_last7_days,
        }
    }
}
