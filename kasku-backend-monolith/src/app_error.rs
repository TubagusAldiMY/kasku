use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AppError {
    #[error("not found")]
    NotFound,

    #[error("unauthorized: {0}")]
    Unauthorized(String),

    #[error("forbidden")]
    Forbidden,

    #[error("validation error: {0}")]
    Validation(String),

    #[error("conflict: {0}")]
    Conflict(String),

    #[error("unprocessable: {0}")]
    Unprocessable(String),

    #[error("rate limit exceeded")]
    RateLimited,

    #[error("tenant not provisioned")]
    TenantNotProvisioned,

    #[error("tier limit exceeded: {0}")]
    TierLimitExceeded(String),

    #[error("external service error: {0}")]
    ExternalService(String),

    #[error("internal error")]
    Internal(#[from] anyhow::Error),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, code, message) = match &self {
            AppError::NotFound => (
                StatusCode::NOT_FOUND,
                "NOT_FOUND",
                "Resource tidak ditemukan.".to_string(),
            ),
            AppError::Unauthorized(msg) => (
                StatusCode::UNAUTHORIZED,
                "UNAUTHORIZED",
                msg.clone(),
            ),
            AppError::Forbidden => (
                StatusCode::FORBIDDEN,
                "FORBIDDEN",
                "Akses ditolak.".to_string(),
            ),
            AppError::Validation(msg) => (
                StatusCode::UNPROCESSABLE_ENTITY,
                "VALIDATION_ERROR",
                msg.clone(),
            ),
            AppError::Conflict(msg) => (
                StatusCode::CONFLICT,
                "CONFLICT",
                msg.clone(),
            ),
            AppError::Unprocessable(msg) => (
                StatusCode::UNPROCESSABLE_ENTITY,
                "UNPROCESSABLE",
                msg.clone(),
            ),
            AppError::RateLimited => (
                StatusCode::TOO_MANY_REQUESTS,
                "RATE_LIMIT_EXCEEDED",
                "Terlalu banyak request. Coba lagi nanti.".to_string(),
            ),
            AppError::TenantNotProvisioned => (
                StatusCode::FORBIDDEN,
                "TENANT_NOT_PROVISIONED",
                "Akun belum diinisialisasi. Silakan hubungi support.".to_string(),
            ),
            AppError::TierLimitExceeded(msg) => (
                StatusCode::PAYMENT_REQUIRED,
                "TIER_LIMIT_EXCEEDED",
                msg.clone(),
            ),
            AppError::ExternalService(msg) => (
                StatusCode::BAD_GATEWAY,
                "EXTERNAL_SERVICE_ERROR",
                msg.clone(),
            ),
            AppError::Internal(e) => {
                tracing::error!(error = %e, "internal server error");
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "INTERNAL_ERROR",
                    "Terjadi kesalahan internal. Silakan coba lagi.".to_string(),
                )
            }
        };

        let body = json!({
            "success": false,
            "error": {
                "code": code,
                "message": message
            }
        });

        (status, Json(body)).into_response()
    }
}

impl From<sqlx::Error> for AppError {
    fn from(e: sqlx::Error) -> Self {
        AppError::Internal(anyhow::anyhow!(e))
    }
}
