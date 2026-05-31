use std::sync::Arc;
use axum::{
    extract::{Path, State},
    response::{IntoResponse, Response},
};

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::response::ApiResponse;
use crate::modules::price::domain::error::PriceError;

use super::dto::PriceResponse;

/// GET /prices/:symbol — public endpoint, no auth required
pub async fn get_price(
    State(state): State<Arc<AppState>>,
    Path(symbol): Path<String>,
) -> Response {
    match state.price_uc.execute(&symbol, "").await {
        Ok(result) => ApiResponse::ok(PriceResponse::from(result)).into_response(),
        Err(PriceError::PriceNotFound(_)) => AppError::NotFound.into_response(),
        Err(PriceError::UnsupportedSource(s)) => {
            AppError::Validation(format!("sumber tidak didukung: {}", s)).into_response()
        }
        Err(e) => {
            tracing::error!(error = %e, "price fetch error");
            AppError::Internal(anyhow::anyhow!(e.to_string())).into_response()
        }
    }
}
