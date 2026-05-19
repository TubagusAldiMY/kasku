use axum::{
    extract::{Query, State},
    http::{header, HeaderMap, StatusCode},
    response::IntoResponse,
    Json,
};
use chrono::{DateTime, Utc};
use serde::Deserialize;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

use crate::domain::entity::SyncOperation;
use crate::domain::error::DomainError;
use crate::usecase::pull_sync::PullSyncUseCase;
use crate::usecase::push_sync::PushSyncUseCase;

/// Counters exposed via /metrics (Prometheus text format).
#[derive(Default)]
pub struct SyncMetrics {
    pub push_total: AtomicU64,
    pub push_conflicts_total: AtomicU64,
    pub push_skipped_total: AtomicU64,
    pub pull_total: AtomicU64,
    pub errors_total: AtomicU64,
}

/// Shared application state.
pub struct AppState {
    pub push_uc: PushSyncUseCase,
    pub pull_uc: PullSyncUseCase,
    pub service_version: String,
    pub db_pool: sqlx::PgPool,
    pub metrics: SyncMetrics,
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
/// Response body kept generic — no upstream library trace or internal address leaks.
fn domain_error_response(err: DomainError) -> impl IntoResponse {
    let (status, code, message) = match &err {
        DomainError::Unauthorized => (
            StatusCode::UNAUTHORIZED,
            "UNAUTHORIZED",
            "header autentikasi tidak ditemukan".to_string(),
        ),
        DomainError::InvalidTenantSchema(s) => (
            StatusCode::BAD_REQUEST,
            "INVALID_TENANT_SCHEMA",
            format!("tenant schema tidak valid: {s}"),
        ),
        DomainError::TenantMismatch { .. } => (
            StatusCode::FORBIDDEN,
            "TENANT_MISMATCH",
            "tenant schema tidak cocok dengan user".to_string(),
        ),
        DomainError::UnsupportedEntityType(t) => (
            StatusCode::BAD_REQUEST,
            "UNSUPPORTED_ENTITY_TYPE",
            format!("tipe entitas tidak didukung: {t}"),
        ),
        DomainError::UpstreamTimeout => (
            StatusCode::GATEWAY_TIMEOUT,
            "UPSTREAM_TIMEOUT",
            "upstream service timeout".to_string(),
        ),
        DomainError::UpstreamUnavailable => (
            StatusCode::BAD_GATEWAY,
            "UPSTREAM_UNAVAILABLE",
            "upstream service unavailable".to_string(),
        ),
        DomainError::UpstreamInvalidResponse => (
            StatusCode::BAD_GATEWAY,
            "UPSTREAM_INVALID_RESPONSE",
            "upstream response invalid".to_string(),
        ),
        DomainError::TenantNotProvisioned(schema) => (
            StatusCode::SERVICE_UNAVAILABLE,
            "TENANT_NOT_PROVISIONED",
            format!("tenant '{schema}' belum di-provisioning, coba beberapa saat lagi"),
        ),
        DomainError::DatabaseError(_) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            "INTERNAL_ERROR",
            "internal server error".to_string(),
        ),
    };

    (
        status,
        Json(serde_json::json!({
            "success": false,
            "error": { "code": code, "message": message }
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

/// GET /metrics — Prometheus text format.
pub async fn metrics(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    let m = &state.metrics;
    let body = format!(
        "# HELP kasku_service_info KasKu service metadata\n\
         # TYPE kasku_service_info gauge\n\
         kasku_service_info{{service=\"sync-service\",version=\"{ver}\"}} 1\n\
         # HELP kasku_sync_push_total Jumlah operasi push yang berhasil di-apply\n\
         # TYPE kasku_sync_push_total counter\n\
         kasku_sync_push_total {push}\n\
         # HELP kasku_sync_push_conflicts_total Jumlah operasi push yang resolve via Server Wins\n\
         # TYPE kasku_sync_push_conflicts_total counter\n\
         kasku_sync_push_conflicts_total {conflicts}\n\
         # HELP kasku_sync_push_skipped_total Jumlah operasi push yang skipped (idempotency hit)\n\
         # TYPE kasku_sync_push_skipped_total counter\n\
         kasku_sync_push_skipped_total {skipped}\n\
         # HELP kasku_sync_pull_total Jumlah request pull yang sukses\n\
         # TYPE kasku_sync_pull_total counter\n\
         kasku_sync_pull_total {pull}\n\
         # HELP kasku_sync_errors_total Jumlah request yang berakhir error\n\
         # TYPE kasku_sync_errors_total counter\n\
         kasku_sync_errors_total {errors}\n",
        ver = state.service_version,
        push = m.push_total.load(Ordering::Relaxed),
        conflicts = m.push_conflicts_total.load(Ordering::Relaxed),
        skipped = m.push_skipped_total.load(Ordering::Relaxed),
        pull = m.pull_total.load(Ordering::Relaxed),
        errors = m.errors_total.load(Ordering::Relaxed),
    );

    (
        [(header::CONTENT_TYPE, "text/plain; version=0.0.4")],
        body,
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
        Err(err) => {
            state.metrics.errors_total.fetch_add(1, Ordering::Relaxed);
            return domain_error_response(err).into_response();
        }
    };

    match state
        .push_uc
        .execute(&user_id, &tenant_schema, body.operations)
        .await
    {
        Ok(result) => {
            state
                .metrics
                .push_total
                .fetch_add(result.processed as u64, Ordering::Relaxed);
            state
                .metrics
                .push_conflicts_total
                .fetch_add(result.conflicts as u64, Ordering::Relaxed);
            state
                .metrics
                .push_skipped_total
                .fetch_add(result.skipped as u64, Ordering::Relaxed);
            (
                StatusCode::OK,
                Json(serde_json::json!({"success": true, "data": result})),
            )
                .into_response()
        }
        Err(err) => {
            state.metrics.errors_total.fetch_add(1, Ordering::Relaxed);
            domain_error_response(err).into_response()
        }
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
        Err(err) => {
            state.metrics.errors_total.fetch_add(1, Ordering::Relaxed);
            return domain_error_response(err).into_response();
        }
    };

    match state
        .pull_uc
        .execute(&user_id, &tenant_schema, query.since)
        .await
    {
        Ok(result) => {
            state.metrics.pull_total.fetch_add(1, Ordering::Relaxed);
            (
                StatusCode::OK,
                Json(serde_json::json!({"success": true, "data": result})),
            )
                .into_response()
        }
        Err(err) => {
            state.metrics.errors_total.fetch_add(1, Ordering::Relaxed);
            domain_error_response(err).into_response()
        }
    }
}
