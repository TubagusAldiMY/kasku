use std::sync::Arc;
use axum::{extract::{Extension, State}, response::{IntoResponse, Response}};

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::response::ApiResponse;

use super::dto::ProfileResponse;
use crate::modules::user::domain::error::UserError;

fn user_err(e: UserError) -> AppError {
    match e {
        UserError::ProfileNotFound => AppError::NotFound,
        UserError::Internal(inner) => AppError::Internal(inner),
        UserError::ProvisioningFailed(s) => AppError::Internal(anyhow::anyhow!(s)),
    }
}

pub async fn get_profile(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
) -> Result<Response, AppError> {
    let profile = state.user_uc.get_profile.execute(claims.user_id)
        .await
        .map_err(user_err)?;

    Ok(ApiResponse::ok(ProfileResponse {
        user_id: profile.user_id.to_string(),
        email: profile.email,
        username: profile.username,
        display_name: profile.display_name,
    }).into_response())
}
