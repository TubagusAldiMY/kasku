use std::sync::Arc;
use axum::{
    extract::{Extension, Path, State},
    response::{IntoResponse, Response},
    Json,
};
use chrono::Utc;
use uuid::Uuid;

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::middleware::tenant_inject::TenantContext;
use crate::shared::response::{created, no_content, ApiResponse};

use super::dto::*;

pub async fn list_assets(
    State(state): State<Arc<AppState>>,
    Extension(_claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
) -> Result<Response, AppError> {
    let assets = state.investment_uc.list_assets(&tenant.schema).await?;
    Ok(ApiResponse::ok(assets.into_iter().map(AssetResponse::from).collect::<Vec<_>>()).into_response())
}

pub async fn get_asset(
    State(state): State<Arc<AppState>>,
    Extension(_claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    let asset = state.investment_uc.get_asset(&tenant.schema, id).await?;
    Ok(ApiResponse::ok(AssetResponse::from(asset)).into_response())
}

pub async fn create_asset(
    State(state): State<Arc<AppState>>,
    Extension(_claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Json(req): Json<CreateAssetRequest>,
) -> Result<Response, AppError> {
    if req.name.trim().is_empty() {
        return Err(AppError::Validation("name tidak boleh kosong".into()));
    }
    let asset = state.investment_uc.create_asset(
        &tenant.schema, req.name, req.asset_type, req.symbol,
        req.quantity, req.avg_buy_price, req.currency, req.sort_order,
    ).await?;
    Ok(created(AssetResponse::from(asset)))
}

pub async fn update_asset(
    State(state): State<Arc<AppState>>,
    Extension(_claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
    Json(req): Json<UpdateAssetRequest>,
) -> Result<Response, AppError> {
    let asset = state.investment_uc.update_asset(
        &tenant.schema, id,
        req.name, req.asset_type, req.symbol,
        req.quantity, req.avg_buy_price, req.currency, req.sort_order,
    ).await?;
    Ok(ApiResponse::ok(AssetResponse::from(asset)).into_response())
}

pub async fn delete_asset(
    State(state): State<Arc<AppState>>,
    Extension(_claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    state.investment_uc.delete_asset(&tenant.schema, id).await?;
    Ok(no_content())
}

pub async fn record_unit_transaction(
    State(state): State<Arc<AppState>>,
    Extension(_claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(asset_id): Path<Uuid>,
    Json(req): Json<RecordTransactionRequest>,
) -> Result<Response, AppError> {
    let date = chrono::DateTime::parse_from_rfc3339(&req.transaction_date)
        .map(|dt| dt.with_timezone(&Utc))
        .map_err(|_| AppError::Validation("format transaction_date tidak valid (gunakan RFC3339)".into()))?;
    let h = state.investment_uc.record_unit_transaction(
        &tenant.schema, asset_id,
        req.transaction_type, req.quantity_change, req.price_per_unit,
        req.notes, date,
    ).await?;
    Ok(created(UnitHistoryResponse::from(h)))
}

pub async fn get_unit_history(
    State(state): State<Arc<AppState>>,
    Extension(_claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(asset_id): Path<Uuid>,
) -> Result<Response, AppError> {
    let history = state.investment_uc.get_unit_history(&tenant.schema, asset_id).await?;
    Ok(ApiResponse::ok(history.into_iter().map(UnitHistoryResponse::from).collect::<Vec<_>>()).into_response())
}
