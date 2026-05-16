use axum::{
    extract::{Path, Query, State},
    http::{header, StatusCode},
    response::IntoResponse,
    Json,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;

use crate::AppState;

/// Health check response.
#[derive(Serialize)]
struct HealthResponse {
    status: String,
    version: String,
    checks: HealthChecks,
}

#[derive(Serialize)]
struct HealthChecks {
    postgres: String,
}

/// GET /health — Health check endpoint.
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
        Json(HealthResponse {
            status: overall.to_string(),
            version: state.service_version.clone(),
            checks: HealthChecks {
                postgres: pg_status.to_string(),
            },
        }),
    )
}

/// GET /metrics — minimal Prometheus scrape endpoint.
pub async fn metrics() -> impl IntoResponse {
    (
		[(header::CONTENT_TYPE, "text/plain; version=0.0.4")],
		"# HELP kasku_service_info KasKu service metadata\n# TYPE kasku_service_info gauge\nkasku_service_info{service=\"price-service\"} 1\n",
	)
}

/// Price response for REST API.
#[derive(Serialize)]
struct PriceResponse {
    success: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    data: Option<PriceData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<ErrorData>,
}

#[derive(Serialize)]
struct PriceData {
    symbol: String,
    price_idr: f64,
    price_usd: f64,
    is_fresh: bool,
    updated_at: String,
}

#[derive(Serialize)]
struct ErrorData {
    code: String,
    message: String,
}

/// Query parameters for price endpoint.
#[derive(Deserialize)]
pub struct PriceQuery {
    #[serde(default)]
    source: String,
}

/// GET /v1/prices/:symbol — Get price for a symbol.
pub async fn get_price(
    State(state): State<Arc<AppState>>,
    Path(symbol): Path<String>,
    Query(query): Query<PriceQuery>,
) -> impl IntoResponse {
    match state.get_price_uc.execute(&symbol, &query.source).await {
        Ok(result) => (
            StatusCode::OK,
            Json(PriceResponse {
                success: true,
                data: Some(PriceData {
                    symbol: result.symbol,
                    price_idr: result.price_idr,
                    price_usd: result.price_usd,
                    is_fresh: result.is_fresh,
                    updated_at: result.updated_at.to_rfc3339(),
                }),
                error: None,
            }),
        ),
        Err(err) => {
            let (status, code) = match &err {
                crate::domain::error::DomainError::PriceNotFound(_) => {
                    (StatusCode::NOT_FOUND, "PRICE_NOT_FOUND")
                }
                crate::domain::error::DomainError::SsrfBlocked(_) => {
                    (StatusCode::FORBIDDEN, "SSRF_BLOCKED")
                }
                _ => (StatusCode::INTERNAL_SERVER_ERROR, "INTERNAL_ERROR"),
            };

            (
                status,
                Json(PriceResponse {
                    success: false,
                    data: None,
                    error: Some(ErrorData {
                        code: code.to_string(),
                        message: err.to_string(),
                    }),
                }),
            )
        }
    }
}
