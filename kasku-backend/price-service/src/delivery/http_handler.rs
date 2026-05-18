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

/// GET /metrics — Prometheus text format.
pub async fn metrics(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    use std::sync::atomic::Ordering;
    let m = &state.metrics;
    let body = format!(
        "# HELP kasku_service_info KasKu service metadata\n\
         # TYPE kasku_service_info gauge\n\
         kasku_service_info{{service=\"price-service\",version=\"{ver}\"}} 1\n\
         # HELP kasku_price_fetch_success_total Jumlah fetch external sukses (CoinGecko/metals.live).\n\
         # TYPE kasku_price_fetch_success_total counter\n\
         kasku_price_fetch_success_total {fs}\n\
         # HELP kasku_price_fetch_failure_total Jumlah fetch external yang gagal.\n\
         # TYPE kasku_price_fetch_failure_total counter\n\
         kasku_price_fetch_failure_total {ff}\n\
         # HELP kasku_price_cache_hit_total Jumlah request yang dilayani dari cache valid.\n\
         # TYPE kasku_price_cache_hit_total counter\n\
         kasku_price_cache_hit_total {ch}\n\
         # HELP kasku_price_cache_miss_total Jumlah request yang miss cache (perlu fetch external).\n\
         # TYPE kasku_price_cache_miss_total counter\n\
         kasku_price_cache_miss_total {cm}\n\
         # HELP kasku_price_stale_fallback_total Jumlah request yang fallback ke cache stale saat external down.\n\
         # TYPE kasku_price_stale_fallback_total counter\n\
         kasku_price_stale_fallback_total {sf}\n",
        ver = state.service_version,
        fs = m.fetch_success_total.load(Ordering::Relaxed),
        ff = m.fetch_failure_total.load(Ordering::Relaxed),
        ch = m.cache_hit_total.load(Ordering::Relaxed),
        cm = m.cache_miss_total.load(Ordering::Relaxed),
        sf = m.stale_fallback_total.load(Ordering::Relaxed),
    );
    (
        [(header::CONTENT_TYPE, "text/plain; version=0.0.4")],
        body,
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
    use std::sync::atomic::Ordering;
    match state.get_price_uc.execute(&symbol, &query.source).await {
        Ok(result) => {
            // Cache accounting: is_fresh=true berarti dilayani dari cache valid
            // ATAU baru saja di-fetch sukses. Untuk membedakan dengan stale fallback,
            // kita gunakan ini sebagai heuristik: is_fresh=false ⇒ stale fallback.
            if result.is_fresh {
                state.metrics.cache_hit_total.fetch_add(1, Ordering::Relaxed);
            } else {
                state.metrics.stale_fallback_total.fetch_add(1, Ordering::Relaxed);
            }
            (
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
            )
        }
        Err(err) => {
            state.metrics.cache_miss_total.fetch_add(1, Ordering::Relaxed);
            state.metrics.fetch_failure_total.fetch_add(1, Ordering::Relaxed);
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
