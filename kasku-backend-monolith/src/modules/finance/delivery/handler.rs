use std::sync::Arc;
use axum::{
    extract::{Extension, Path, State},
    response::{IntoResponse, Response},
    Json,
};
use uuid::Uuid;

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::middleware::tenant_inject::TenantContext;
use crate::shared::middleware::tier_inject::TierLimits;
use crate::shared::response::{created, no_content, ApiResponse};
use crate::modules::finance::domain::error::FinanceError;

use super::dto::{AccountResponse, BalanceHistoryResponse, CreateAccountRequest, UpdateAccountRequest};

fn finance_err_to_app(e: FinanceError) -> AppError {
    match e {
        FinanceError::AccountNotFound => AppError::NotFound,
        FinanceError::TenantNotProvisioned => AppError::TenantNotProvisioned,
        FinanceError::AccountLimitReached => {
            AppError::TierLimitExceeded("Batas jumlah rekening tercapai. Upgrade paket Anda.".into())
        }
        FinanceError::Internal(e) => AppError::Internal(e),
    }
}

pub async fn list_accounts(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
) -> Result<Response, AppError> {
    let accounts = state
        .finance_uc
        .list_accounts(&tenant.schema, claims.user_id)
        .await
        .map_err(finance_err_to_app)?;

    let resp: Vec<AccountResponse> = accounts.into_iter().map(AccountResponse::from).collect();
    Ok(ApiResponse::ok(resp).into_response())
}

pub async fn create_account(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Extension(limits): Extension<TierLimits>,
    Json(req): Json<CreateAccountRequest>,
) -> Result<Response, AppError> {
    let account = state
        .finance_uc
        .create_account(
            &tenant.schema,
            claims.user_id,
            req.name,
            req.account_type,
            req.currency,
            req.color,
            req.icon,
            req.initial_balance,
            req.is_default,
            &limits,
        )
        .await
        .map_err(finance_err_to_app)?;

    Ok(created(AccountResponse::from(account)))
}

pub async fn update_account(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
    Json(req): Json<UpdateAccountRequest>,
) -> Result<Response, AppError> {
    let account = state
        .finance_uc
        .update_account(
            &tenant.schema,
            id,
            claims.user_id,
            req.name,
            req.color,
            req.icon,
            req.is_default,
        )
        .await
        .map_err(finance_err_to_app)?;

    Ok(ApiResponse::ok(AccountResponse::from(account)).into_response())
}

pub async fn delete_account(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    state
        .finance_uc
        .delete_account(&tenant.schema, id, claims.user_id)
        .await
        .map_err(finance_err_to_app)?;

    Ok(no_content())
}

pub async fn get_balance_history(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    let history = state
        .finance_uc
        .get_balance_history(&tenant.schema, id, claims.user_id)
        .await
        .map_err(finance_err_to_app)?;

    let resp: Vec<BalanceHistoryResponse> = history.into_iter().map(BalanceHistoryResponse::from).collect();
    Ok(ApiResponse::ok(resp).into_response())
}
