use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum AdminRole { SuperAdmin, Support }

impl std::fmt::Display for AdminRole {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self { AdminRole::SuperAdmin => write!(f, "SUPER_ADMIN"), AdminRole::Support => write!(f, "SUPPORT") }
    }
}

impl std::str::FromStr for AdminRole {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s { "SUPER_ADMIN" => Ok(AdminRole::SuperAdmin), "SUPPORT" => Ok(AdminRole::Support), other => Err(format!("unknown role: {}", other)) }
    }
}

#[derive(Debug, Clone)]
pub struct AdminUser {
    pub id: Uuid,
    pub username: String,
    pub password_hash: String,
    pub role: AdminRole,
    pub is_active: bool,
    pub last_login_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl<'r> sqlx::FromRow<'r, sqlx::postgres::PgRow> for AdminUser {
    fn from_row(row: &'r sqlx::postgres::PgRow) -> Result<Self, sqlx::Error> {
        use sqlx::Row;
        let role_str: String = row.try_get("role")?;
        let role = role_str.parse::<AdminRole>().map_err(|e| sqlx::Error::ColumnDecode {
            index: "role".to_string(),
            source: Box::new(std::io::Error::new(std::io::ErrorKind::InvalidData, e)),
        })?;
        Ok(AdminUser {
            id: row.try_get("id")?,
            username: row.try_get("username")?,
            password_hash: row.try_get("password_hash")?,
            role,
            is_active: row.try_get("is_active")?,
            last_login_at: row.try_get("last_login_at")?,
            created_at: row.try_get("created_at")?,
            updated_at: row.try_get("updated_at")?,
        })
    }
}

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct UserSummary {
    pub id: Uuid,
    pub email: String,
    pub username: String,
    pub is_active: bool,
    pub email_verified: bool,
    pub created_at: DateTime<Utc>,
    pub last_login_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Serialize)]
pub struct DashboardStats {
    pub total_users: i64,
    pub total_active_users: i64,
    pub new_users_last7_days: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AuditAction { Login, Logout, SuspendUser, ActivateUser, OverrideSubscription }

impl std::fmt::Display for AuditAction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            AuditAction::Login => write!(f, "LOGIN"),
            AuditAction::Logout => write!(f, "LOGOUT"),
            AuditAction::SuspendUser => write!(f, "SUSPEND_USER"),
            AuditAction::ActivateUser => write!(f, "ACTIVATE_USER"),
            AuditAction::OverrideSubscription => write!(f, "OVERRIDE_SUBSCRIPTION"),
        }
    }
}

#[derive(Debug, Clone)]
pub struct AuditLogEntry {
    pub id: Uuid,
    pub admin_id: Uuid,
    pub action: AuditAction,
    pub target_user_id: Option<Uuid>,
    pub target_entity: Option<String>,
    pub metadata: serde_json::Value,
    pub success: bool,
    pub created_at: DateTime<Utc>,
}

impl<'r> sqlx::FromRow<'r, sqlx::postgres::PgRow> for AuditLogEntry {
    fn from_row(row: &'r sqlx::postgres::PgRow) -> Result<Self, sqlx::Error> {
        use sqlx::Row;
        let action_str: String = row.try_get("action")?;
        let action = match action_str.as_str() {
            "LOGIN" => AuditAction::Login,
            "LOGOUT" => AuditAction::Logout,
            "SUSPEND_USER" => AuditAction::SuspendUser,
            "ACTIVATE_USER" => AuditAction::ActivateUser,
            "OVERRIDE_SUBSCRIPTION" => AuditAction::OverrideSubscription,
            _ => AuditAction::Login,
        };
        Ok(AuditLogEntry {
            id: row.try_get("id")?,
            admin_id: row.try_get("admin_id")?,
            action,
            target_user_id: row.try_get("target_user_id")?,
            target_entity: row.try_get("target_entity")?,
            metadata: row.try_get::<serde_json::Value, _>("metadata").unwrap_or(serde_json::Value::Null),
            success: row.try_get("success")?,
            created_at: row.try_get("created_at")?,
        })
    }
}
