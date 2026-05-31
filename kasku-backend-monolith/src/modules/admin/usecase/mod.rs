use std::sync::Arc;
use chrono::Utc;
use jsonwebtoken::{encode, Algorithm, EncodingKey, Header};
use uuid::Uuid;

use crate::app_error::AppError;
use crate::modules::admin::domain::entity::{AdminRole, AdminUser, AuditAction, AuditLogEntry, DashboardStats, UserSummary};
use crate::modules::admin::domain::error::AdminError;
use crate::modules::admin::domain::repository::{AdminUserReadRepository, AdminUserRepository, AuditLogRepository};
use crate::modules::auth::usecase::helpers::{hash_password, verify_password, Argon2Config};
use crate::shared::middleware::admin_auth::AdminClaims;

pub struct AdminUseCases {
    pub admin_repo: Arc<dyn AdminUserRepository>,
    pub user_read_repo: Arc<dyn AdminUserReadRepository>,
    pub audit_repo: Arc<dyn AuditLogRepository>,
    jwt_secret: String,
    jwt_ttl_seconds: u64,
    argon2: Argon2Config,
}

impl AdminUseCases {
    pub fn new(
        admin_repo: Arc<dyn AdminUserRepository>,
        user_read_repo: Arc<dyn AdminUserReadRepository>,
        audit_repo: Arc<dyn AuditLogRepository>,
        jwt_secret: String,
        jwt_ttl_seconds: u64,
        argon2: Argon2Config,
    ) -> Self {
        Self { admin_repo, user_read_repo, audit_repo, jwt_secret, jwt_ttl_seconds, argon2 }
    }

    pub async fn login(&self, username: &str, password: &str) -> Result<(String, AdminUser), AdminError> {
        let admin = self.admin_repo.find_by_username(username).await?
            .ok_or(AdminError::InvalidCredentials)?;

        if !admin.is_active { return Err(AdminError::AdminInactive); }

        let ok = verify_password(password, &admin.password_hash)
            .map_err(|e| AdminError::Internal(e.to_string()))?;
        if !ok { return Err(AdminError::InvalidCredentials); }

        let token = self.generate_token(&admin)?;
        self.admin_repo.update_last_login(admin.id).await?;

        let entry = AuditLogEntry {
            id: Uuid::new_v4(),
            admin_id: admin.id,
            action: AuditAction::Login,
            target_user_id: None,
            target_entity: None,
            metadata: serde_json::json!({"username": username}),
            success: true,
            created_at: Utc::now(),
        };
        let _ = self.audit_repo.create(&entry).await;

        Ok((token, admin))
    }

    pub async fn get_current(&self, admin_id: Uuid) -> Result<AdminUser, AdminError> {
        self.admin_repo.find_by_id(admin_id).await?.ok_or(AdminError::AdminNotFound)
    }

    pub async fn list_users(
        &self, page: i64, per_page: i64, search: Option<&str>,
    ) -> Result<(Vec<UserSummary>, i64), AdminError> {
        self.user_read_repo.list_users(page, per_page, search).await
    }

    pub async fn get_user_detail(&self, id: Uuid) -> Result<UserSummary, AdminError> {
        self.user_read_repo.find_user_by_id(id).await?.ok_or(AdminError::UserNotFound)
    }

    pub async fn suspend_user(&self, admin_id: Uuid, user_id: Uuid) -> Result<(), AdminError> {
        let _ = self.user_read_repo.find_user_by_id(user_id).await?.ok_or(AdminError::UserNotFound)?;
        self.user_read_repo.suspend_user(user_id).await?;
        let entry = AuditLogEntry {
            id: Uuid::new_v4(),
            admin_id,
            action: AuditAction::SuspendUser,
            target_user_id: Some(user_id),
            target_entity: Some("user".into()),
            metadata: serde_json::json!({}),
            success: true,
            created_at: Utc::now(),
        };
        let _ = self.audit_repo.create(&entry).await;
        Ok(())
    }

    pub async fn activate_user(&self, admin_id: Uuid, user_id: Uuid) -> Result<(), AdminError> {
        let _ = self.user_read_repo.find_user_by_id(user_id).await?.ok_or(AdminError::UserNotFound)?;
        self.user_read_repo.activate_user(user_id).await?;
        let entry = AuditLogEntry {
            id: Uuid::new_v4(),
            admin_id,
            action: AuditAction::ActivateUser,
            target_user_id: Some(user_id),
            target_entity: Some("user".into()),
            metadata: serde_json::json!({}),
            success: true,
            created_at: Utc::now(),
        };
        let _ = self.audit_repo.create(&entry).await;
        Ok(())
    }

    pub async fn dashboard_stats(&self) -> Result<DashboardStats, AdminError> {
        self.user_read_repo.dashboard_stats().await
    }

    pub async fn list_audit_log(
        &self, page: i64, per_page: i64, admin_id: Option<Uuid>,
    ) -> Result<(Vec<AuditLogEntry>, i64), AdminError> {
        self.audit_repo.list(page, per_page, admin_id).await
    }

    pub async fn bootstrap(&self, username: &str, password: &str) -> Result<bool, AdminError> {
        let count = self.admin_repo.count().await?;
        if count > 0 { return Ok(false); }

        let hash = hash_password(password, &self.argon2)
            .map_err(|e| AdminError::Internal(e.to_string()))?;
        let now = Utc::now();
        let admin = AdminUser {
            id: Uuid::new_v4(),
            username: username.to_string(),
            password_hash: hash,
            role: AdminRole::SuperAdmin,
            is_active: true,
            last_login_at: None,
            created_at: now,
            updated_at: now,
        };
        self.admin_repo.create(&admin).await?;
        Ok(true)
    }

    fn generate_token(&self, admin: &AdminUser) -> Result<String, AdminError> {
        let now = Utc::now().timestamp();
        let exp = now + self.jwt_ttl_seconds as i64;
        let claims = AdminClaims {
            sub: admin.id.to_string(),
            username: admin.username.clone(),
            role: admin.role.to_string(),
            exp,
            iat: now,
        };
        let key = EncodingKey::from_secret(self.jwt_secret.as_bytes());
        encode(&Header::new(Algorithm::HS256), &claims, &key)
            .map_err(|e| AdminError::Internal(format!("token sign error: {}", e)))
    }
}

pub fn admin_err_to_app(e: AdminError) -> AppError {
    match e {
        AdminError::InvalidCredentials => AppError::Unauthorized("kredensial tidak valid".into()),
        AdminError::AdminInactive => AppError::Forbidden,
        AdminError::AdminNotFound => AppError::NotFound,
        AdminError::UserNotFound => AppError::NotFound,
        AdminError::SubscriptionNotFound => AppError::NotFound,
        AdminError::PlanNotFound => AppError::NotFound,
        AdminError::InvalidToken => AppError::Unauthorized("token tidak valid".into()),
        AdminError::Unauthorized => AppError::Unauthorized("tidak diotorisasi".into()),
        AdminError::Forbidden => AppError::Forbidden,
        AdminError::Validation(s) => AppError::Validation(s),
        AdminError::Internal(s) => AppError::Internal(anyhow::anyhow!(s)),
    }
}
