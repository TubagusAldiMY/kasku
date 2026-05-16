use axum::{
    extract::{Query, State},
    http::{header, HeaderMap, StatusCode},
    response::IntoResponse,
    Json,
};
use chrono::{DateTime, Utc};
use serde::Deserialize;
use std::sync::Arc;

use crate::domain::entity::SyncOperation;
use crate::domain::error::DomainError;
use crate::usecase::pull_sync::PullSyncUseCase;
use crate::usecase::push_sync::PushSyncUseCase;

/// Shared application state.
pub struct AppState {
    pub push_uc: PushSyncUseCase,
    pub pull_uc: PullSyncUseCase,
    pub service_version: String,
    pub db_pool: sqlx::PgPool,
}

// ── Header constants ────────────────────────────────────────────────────
const HEADER_USER_ID: &str = "x-user-id";
const HEADER_TENANT_SCHEMA: &str = "x-tenant-schema";

/// Extract user_id and tenant_schema from gateway-injected headers.
fn extract_request_context(headers: &HeaderMap) -> Result<(String, String), DomainError> {
    let user_id = headers
        .get(HEADER_USER_ID)
        .and_then(|v| v.to_str().ok())
        .ok_or(DomainError::Unauthorized)?
        .to_string();

    let tenant_schema = headers
        .get(HEADER_TENANT_SCHEMA)
        .and_then(|v| v.to_str().ok())
        .ok_or(DomainError::Unauthorized)?
        .to_string();

    Ok((user_id, tenant_schema))
}

/// Map domain errors to HTTP responses.
fn domain_error_response(err: DomainError) -> impl IntoResponse {
    let (status, code) = match &err {
        DomainError::Unauthorized => (StatusCode::UNAUTHORIZED, "UNAUTHORIZED"),
        DomainError::InvalidTenantSchema(_) => (StatusCode::BAD_REQUEST, "INVALID_TENANT_SCHEMA"),
        DomainError::TenantMismatch { .. } => (StatusCode::FORBIDDEN, "TENANT_MISMATCH"),
        DomainError::UnsupportedEntityType(_) => {
            (StatusCode::BAD_REQUEST, "UNSUPPORTED_ENTITY_TYPE")
        }
        _ => (StatusCode::INTERNAL_SERVER_ERROR, "INTERNAL_ERROR"),
    };

    (
        status,
        Json(serde_json::json!({
            "success": false,
            "error": {
                "code": code,
                "message": err.to_string()
            }
        })),
    )
}

/// GET /health
pub async fn health(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    let pg_status = match crate::infrastructure::db::ping(&state.db_pool).await {
        Ok(_) => "healthy",
        Err(_) => "unhealthy",
    };

    let overall = if pg_status == "healthy" {
        "healthy"
    } else {
        "degraded"
    };
    let status_code = if overall == "healthy" {
        StatusCode::OK
    } else {
        StatusCode::SERVICE_UNAVAILABLE
    };

    (
        status_code,
        Json(serde_json::json!({
            "status": overall,
            "version": state.service_version,
            "checks": { "postgres": pg_status }
        })),
    )
}

/// GET /metrics — minimal Prometheus scrape endpoint.
pub async fn metrics() -> impl IntoResponse {
    (
		[(header::CONTENT_TYPE, "text/plain; version=0.0.4")],
		"# HELP kasku_service_info KasKu service metadata\n# TYPE kasku_service_info gauge\nkasku_service_info{service=\"sync-service\"} 1\n",
	)
}

/// Push request body.
#[derive(Deserialize)]
pub struct PushRequest {
    pub operations: Vec<SyncOperation>,
}

/// POST /v1/sync/push — Batch push offline operations.
pub async fn push_sync(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(body): Json<PushRequest>,
) -> impl IntoResponse {
    let (user_id, tenant_schema) = match extract_request_context(&headers) {
        Ok(ctx) => ctx,
        Err(err) => return domain_error_response(err).into_response(),
    };

    match state
        .push_uc
        .execute(&user_id, &tenant_schema, body.operations)
        .await
    {
        Ok(result) => (
            StatusCode::OK,
            Json(serde_json::json!({"success": true, "data": result})),
        )
            .into_response(),
        Err(err) => domain_error_response(err).into_response(),
    }
}

/// Query parameters for pull sync.
#[derive(Deserialize)]
pub struct PullQuery {
    pub since: DateTime<Utc>,
}

/// GET /v1/sync/pull?since={timestamp} — Pull changes since timestamp.
pub async fn pull_sync(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Query(query): Query<PullQuery>,
) -> impl IntoResponse {
    let (user_id, tenant_schema) = match extract_request_context(&headers) {
        Ok(ctx) => ctx,
        Err(err) => return domain_error_response(err).into_response(),
    };

    match state
        .pull_uc
        .execute(&user_id, &tenant_schema, query.since)
        .await
    {
        Ok(result) => (
            StatusCode::OK,
            Json(serde_json::json!({"success": true, "data": result})),
        )
            .into_response(),
        Err(err) => domain_error_response(err).into_response(),
    }
}
