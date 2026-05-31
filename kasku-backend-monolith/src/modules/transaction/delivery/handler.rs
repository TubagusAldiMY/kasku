use std::sync::Arc;
use axum::{
    extract::{Extension, Path, Query, State},
    http::header,
    response::{IntoResponse, Response},
    Json,
};
use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::middleware::tenant_inject::TenantContext;
use crate::shared::middleware::tier_inject::TierLimits;
use crate::shared::response::{created, no_content, ApiResponse};
use crate::modules::transaction::domain::entity::{BudgetPeriodType, CategoryType, TransactionType};

use super::dto::*;

fn parse_datetime(s: &str) -> Result<DateTime<Utc>, AppError> {
    DateTime::parse_from_rfc3339(s)
        .map(|dt| dt.with_timezone(&Utc))
        .map_err(|_| AppError::Validation(format!("format tanggal tidak valid: {}", s)))
}

fn parse_transaction_type(s: &str) -> Result<TransactionType, AppError> {
    s.parse::<TransactionType>()
        .map_err(|e| AppError::Validation(e))
}

fn parse_category_type(s: &str) -> Result<CategoryType, AppError> {
    s.parse::<CategoryType>()
        .map_err(|e| AppError::Validation(e))
}

fn parse_budget_period_type(s: &str) -> Result<BudgetPeriodType, AppError> {
    s.parse::<BudgetPeriodType>()
        .map_err(|e| AppError::Validation(e))
}

fn default_from() -> DateTime<Utc> {
    Utc::now() - chrono::Duration::days(30)
}

fn default_to() -> DateTime<Utc> {
    Utc::now()
}

// ---------- Transaction handlers ----------

pub async fn list_transactions(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Query(q): Query<ListTransactionsQuery>,
) -> Result<Response, AppError> {
    let from = q.from.as_deref().map(parse_datetime).transpose()?.unwrap_or_else(default_from);
    let to = q.to.as_deref().map(parse_datetime).transpose()?.unwrap_or_else(default_to);

    let txs = state
        .transaction_uc
        .list_transactions(&tenant.schema, claims.user_id, from, to)
        .await?;

    let resp: Vec<TransactionResponse> = txs.into_iter().map(TransactionResponse::from).collect();
    Ok(ApiResponse::ok(resp).into_response())
}

pub async fn create_transaction(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Extension(limits): Extension<TierLimits>,
    Json(req): Json<CreateTransactionRequest>,
) -> Result<Response, AppError> {
    let transaction_type = parse_transaction_type(&req.transaction_type)?;
    let transaction_date = parse_datetime(&req.transaction_date)?;

    let tx = state
        .transaction_uc
        .create_transaction(
            &tenant.schema,
            claims.user_id,
            req.account_id,
            req.category_id,
            req.budget_id,
            transaction_type,
            req.amount_idr,
            transaction_date,
            req.notes.unwrap_or_default(),
            req.to_account_id,
            req.sync_id,
            &limits,
        )
        .await?;

    Ok(created(TransactionResponse::from(tx)))
}

pub async fn get_transaction(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    let tx = state
        .transaction_uc
        .get_transaction(&tenant.schema, id, claims.user_id)
        .await?;

    Ok(ApiResponse::ok(TransactionResponse::from(tx)).into_response())
}

pub async fn update_transaction(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
    Json(req): Json<UpdateTransactionRequest>,
) -> Result<Response, AppError> {
    let transaction_type = req.transaction_type.as_deref().map(parse_transaction_type).transpose()?;
    let transaction_date = req.transaction_date.as_deref().map(parse_datetime).transpose()?;

    let tx = state
        .transaction_uc
        .update_transaction(
            &tenant.schema,
            claims.user_id,
            id,
            req.account_id,
            req.category_id,
            req.budget_id,
            transaction_type,
            req.amount_idr,
            transaction_date,
            req.notes,
            req.to_account_id,
        )
        .await?;

    Ok(ApiResponse::ok(TransactionResponse::from(tx)).into_response())
}

pub async fn delete_transaction(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    state
        .transaction_uc
        .delete_transaction(&tenant.schema, id, claims.user_id)
        .await?;
    Ok(no_content())
}

pub async fn get_summary(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Query(q): Query<ListTransactionsQuery>,
) -> Result<Response, AppError> {
    let from = q.from.as_deref().map(parse_datetime).transpose()?.unwrap_or_else(default_from);
    let to = q.to.as_deref().map(parse_datetime).transpose()?.unwrap_or_else(default_to);

    let summary = state
        .transaction_uc
        .get_summary(&tenant.schema, claims.user_id, from, to)
        .await?;

    Ok(ApiResponse::ok(TransactionSummaryResponse::from(summary)).into_response())
}

pub async fn export_csv(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Extension(limits): Extension<TierLimits>,
    Query(q): Query<ListTransactionsQuery>,
) -> Result<Response, AppError> {
    let from = q.from.as_deref().map(parse_datetime).transpose()?;
    let to = q.to.as_deref().map(parse_datetime).transpose()?;

    let csv = state
        .transaction_uc
        .export_csv(&tenant.schema, claims.user_id, from, to, &limits)
        .await?;

    let response = axum::response::Response::builder()
        .status(200)
        .header(header::CONTENT_TYPE, "text/csv")
        .header(header::CONTENT_DISPOSITION, "attachment; filename=\"transactions.csv\"")
        .body(axum::body::Body::from(csv))
        .unwrap();
    Ok(response)
}

// ---------- Category handlers ----------

pub async fn list_categories(
    State(state): State<Arc<AppState>>,
    Extension(tenant): Extension<TenantContext>,
) -> Result<Response, AppError> {
    let cats = state.transaction_uc.list_categories(&tenant.schema).await?;
    let resp: Vec<CategoryResponse> = cats.into_iter().map(CategoryResponse::from).collect();
    Ok(ApiResponse::ok(resp).into_response())
}

pub async fn create_category(
    State(state): State<Arc<AppState>>,
    Extension(tenant): Extension<TenantContext>,
    Json(req): Json<CreateCategoryRequest>,
) -> Result<Response, AppError> {
    let category_type = parse_category_type(&req.category_type)?;
    let cat = state
        .transaction_uc
        .create_category(&tenant.schema, req.name, req.icon, req.color, category_type)
        .await?;
    Ok(created(CategoryResponse::from(cat)))
}

pub async fn update_category(
    State(state): State<Arc<AppState>>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
    Json(req): Json<UpdateCategoryRequest>,
) -> Result<Response, AppError> {
    let category_type = req.category_type.as_deref().map(parse_category_type).transpose()?;
    let cat = state
        .transaction_uc
        .update_category(&tenant.schema, id, req.name, req.icon, req.color, category_type)
        .await?;
    Ok(ApiResponse::ok(CategoryResponse::from(cat)).into_response())
}

pub async fn delete_category(
    State(state): State<Arc<AppState>>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    state.transaction_uc.delete_category(&tenant.schema, id).await?;
    Ok(no_content())
}

// ---------- Budget handlers ----------

pub async fn list_budgets(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
) -> Result<Response, AppError> {
    let budgets = state
        .transaction_uc
        .list_budgets(&tenant.schema, claims.user_id)
        .await?;
    let resp: Vec<BudgetResponse> = budgets.into_iter().map(BudgetResponse::from).collect();
    Ok(ApiResponse::ok(resp).into_response())
}

pub async fn create_budget(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Extension(limits): Extension<TierLimits>,
    Json(req): Json<CreateBudgetRequest>,
) -> Result<Response, AppError> {
    let period_type = parse_budget_period_type(&req.period_type)?;
    let start_date = parse_datetime(&req.start_date)?;
    let end_date = req.end_date.as_deref().map(parse_datetime).transpose()?;

    let bwp = state
        .transaction_uc
        .create_budget(
            &tenant.schema,
            claims.user_id,
            req.name,
            req.limit_idr,
            req.category_id,
            period_type,
            start_date,
            end_date,
            req.alert_threshold,
            req.daily_limit_enabled,
            &limits,
        )
        .await?;

    Ok(created(BudgetResponse::from(bwp)))
}

pub async fn get_budget(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    let bwp = state
        .transaction_uc
        .get_budget(&tenant.schema, id, claims.user_id)
        .await?;
    Ok(ApiResponse::ok(BudgetResponse::from(bwp)).into_response())
}

pub async fn update_budget(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
    Json(req): Json<UpdateBudgetRequest>,
) -> Result<Response, AppError> {
    let period_type = req.period_type.as_deref().map(parse_budget_period_type).transpose()?;
    let start_date = req.start_date.as_deref().map(parse_datetime).transpose()?;
    let end_date = req.end_date.as_deref().map(parse_datetime).transpose()?;

    let bwp = state
        .transaction_uc
        .update_budget(
            &tenant.schema,
            id,
            claims.user_id,
            req.name,
            req.limit_idr,
            req.category_id,
            period_type,
            start_date,
            end_date,
            req.alert_threshold,
            req.daily_limit_enabled,
        )
        .await?;

    Ok(ApiResponse::ok(BudgetResponse::from(bwp)).into_response())
}

pub async fn delete_budget(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Extension(tenant): Extension<TenantContext>,
    Path(id): Path<Uuid>,
) -> Result<Response, AppError> {
    state
        .transaction_uc
        .delete_budget(&tenant.schema, id, claims.user_id)
        .await?;
    Ok(no_content())
}
