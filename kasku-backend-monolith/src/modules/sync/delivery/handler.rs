use std::sync::Arc;
use axum::{
    extract::{Extension, Query, State},
    response::{IntoResponse, Response},
    Json,
};
use chrono::Utc;

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::middleware::tenant_inject::TenantContext;
use crate::shared::response::ApiResponse;
use crate::modules::sync::domain::error::SyncError;

use super::dto::{PullParams, PushRequest};

fn sync_err_to_app(e: SyncError) -> AppError {
    match e {
        SyncError::InvalidTenantSchema(_) | SyncError::TenantMismatch { .. } => {
            AppError::Forbidden
        }
        SyncError::Unauthorized => AppError::Unauthorized("tidak terautentikasi".into()),
        SyncError::TenantNotProvisioned(s) => {
            AppError::TenantNotProvisioned
        }
        SyncError::UnsupportedEntityType(t) => {
            AppError::Validation(format!("tipe entitas tidak didukung: {}", t))
        }
        SyncError::DatabaseError(e) => AppError::Internal(anyhow::anyhow!(e)),
    }
}

pub async fn push_sync(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Json(req): Json<PushRequest>,
) -> Result<Response, AppError> {
    let result = state.sync_uc.push
        .execute(&claims.user_id.to_string(), &tenant.schema, req.operations)
        .await
        .map_err(sync_err_to_app)?;
    Ok(ApiResponse::ok(result).into_response())
}

pub async fn pull_sync(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Query(params): Query<PullParams>,
) -> Result<Response, AppError> {
    let since = params.since
        .as_deref()
        .map(|s| chrono::DateTime::parse_from_rfc3339(s).map(|dt| dt.with_timezone(&Utc)))
        .transpose()
        .map_err(|_| AppError::Validation("format parameter 'since' tidak valid (gunakan RFC3339)".into()))?
        .unwrap_or_else(|| Utc::now() - chrono::Duration::days(30));

    let result = state.sync_uc.pull
        .execute(&claims.user_id.to_string(), &tenant.schema, since)
        .await
        .map_err(sync_err_to_app)?;
    Ok(ApiResponse::ok(result).into_response())
}
