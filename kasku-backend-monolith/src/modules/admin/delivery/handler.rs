use std::sync::Arc;
use axum::{
    extract::{Extension, Path, Query, State},
    Json,
};
use uuid::Uuid;

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::admin_auth::AuthAdmin;
use crate::shared::response::ApiResponse;

use super::dto::*;
use crate::modules::admin::usecase::admin_err_to_app;

pub async fn login(
    State(state): State<Arc<AppState>>,
    Json(body): Json<LoginRequest>,
) -> Result<Json<ApiResponse<LoginResponse>>, AppError> {
    let (token, admin) = state.admin_uc.login(&body.username, &body.password).await
        .map_err(admin_err_to_app)?;
    Ok(Json(ApiResponse::success(LoginResponse {
        token,
        admin: AdminResponse::from(&admin),
    })))
}

pub async fn get_current(
    State(state): State<Arc<AppState>>,
    Extension(auth): Extension<AuthAdmin>,
) -> Result<Json<ApiResponse<AdminResponse>>, AppError> {
    let admin = state.admin_uc.get_current(auth.admin_id).await.map_err(admin_err_to_app)?;
    Ok(Json(ApiResponse::success(AdminResponse::from(&admin))))
}

pub async fn list_users(
    State(state): State<Arc<AppState>>,
    Extension(_auth): Extension<AuthAdmin>,
    Query(q): Query<ListUsersQuery>,
) -> Result<Json<ApiResponse<PaginatedUsersResponse>>, AppError> {
    let (users, total) = state.admin_uc
        .list_users(q.page, q.per_page, q.search.as_deref())
        .await
        .map_err(admin_err_to_app)?;
    Ok(Json(ApiResponse::success(PaginatedUsersResponse {
        users: users.iter().map(UserSummaryResponse::from).collect(),
        total,
        page: q.page,
        per_page: q.per_page,
    })))
}

pub async fn get_user(
    State(state): State<Arc<AppState>>,
    Extension(_auth): Extension<AuthAdmin>,
    Path(user_id): Path<Uuid>,
) -> Result<Json<ApiResponse<UserSummaryResponse>>, AppError> {
    let user = state.admin_uc.get_user_detail(user_id).await.map_err(admin_err_to_app)?;
    Ok(Json(ApiResponse::success(UserSummaryResponse::from(&user))))
}

pub async fn suspend_user(
    State(state): State<Arc<AppState>>,
    Extension(auth): Extension<AuthAdmin>,
    Path(user_id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.admin_uc.suspend_user(auth.admin_id, user_id).await.map_err(admin_err_to_app)?;
    Ok(Json(ApiResponse::success(())))
}

pub async fn activate_user(
    State(state): State<Arc<AppState>>,
    Extension(auth): Extension<AuthAdmin>,
    Path(user_id): Path<Uuid>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.admin_uc.activate_user(auth.admin_id, user_id).await.map_err(admin_err_to_app)?;
    Ok(Json(ApiResponse::success(())))
}

pub async fn dashboard_stats(
    State(state): State<Arc<AppState>>,
    Extension(_auth): Extension<AuthAdmin>,
) -> Result<Json<ApiResponse<DashboardStatsResponse>>, AppError> {
    let stats = state.admin_uc.dashboard_stats().await.map_err(admin_err_to_app)?;
    Ok(Json(ApiResponse::success(DashboardStatsResponse::from(&stats))))
}

pub async fn list_audit_log(
    State(state): State<Arc<AppState>>,
    Extension(_auth): Extension<AuthAdmin>,
    Query(q): Query<ListAuditQuery>,
) -> Result<Json<ApiResponse<PaginatedAuditResponse>>, AppError> {
    let (entries, total) = state.admin_uc
        .list_audit_log(q.page, q.per_page, q.admin_id)
        .await
        .map_err(admin_err_to_app)?;
    Ok(Json(ApiResponse::success(PaginatedAuditResponse {
        entries: entries.iter().map(AuditLogResponse::from).collect(),
        total,
        page: q.page,
        per_page: q.per_page,
    })))
}
