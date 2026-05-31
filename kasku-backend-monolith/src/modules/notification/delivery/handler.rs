use std::sync::Arc;
use axum::{extract::{Extension, State}, response::{IntoResponse, Response}, Json};

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::response::ApiResponse;
use crate::modules::notification::domain::error::NotificationError;
use super::dto::*;

fn notif_err(e: NotificationError) -> AppError {
    match e {
        NotificationError::NotFound => AppError::NotFound,
        NotificationError::Internal(s) => AppError::Internal(anyhow::anyhow!(s)),
    }
}

pub async fn get_preference(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
) -> Result<Response, AppError> {
    let pref = state.notification_uc.get_preference(claims.user_id).await.map_err(notif_err)?;
    Ok(ApiResponse::ok(PreferenceResponse::from(pref)).into_response())
}

pub async fn update_preference(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Json(req): Json<UpdatePreferenceRequest>,
) -> Result<Response, AppError> {
    let pref = state.notification_uc.update_preference(
        claims.user_id,
        req.email_enabled, req.payment_alerts, req.subscription_alerts, req.security_alerts,
    ).await.map_err(notif_err)?;
    Ok(ApiResponse::ok(PreferenceResponse::from(pref)).into_response())
}
