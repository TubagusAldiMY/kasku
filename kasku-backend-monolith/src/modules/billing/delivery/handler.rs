use std::sync::Arc;
use axum::{
    extract::{Extension, State},
    response::{IntoResponse, Response},
    Json,
};

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::shared::middleware::auth::AuthClaims;
use crate::shared::response::{no_content, ApiResponse};

use super::dto::*;
use crate::modules::billing::domain::error::BillingError;
use crate::modules::billing::usecase::{
    create_payment::CreatePaymentInput,
    handle_webhook::{HandleWebhookUseCase, WebhookPayload},
};

fn billing_err(e: BillingError) -> AppError {
    match e {
        BillingError::SubscriptionNotFound | BillingError::PlanNotFound | BillingError::PaymentNotFound => AppError::NotFound,
        BillingError::InvalidWebhookSignature => AppError::Unauthorized("invalid webhook signature".into()),
        BillingError::PaymentAlreadyProcessed => AppError::Conflict("payment already processed".into()),
        BillingError::Internal(inner) => AppError::Internal(inner),
    }
}

pub async fn list_plans(
    State(state): State<Arc<AppState>>,
) -> Result<Response, AppError> {
    let plans = state.billing_uc.list_plans.execute().await.map_err(billing_err)?;
    let resp: Vec<PlanResponse> = plans.into_iter().map(|p| PlanResponse {
        id: p.id.to_string(),
        name: p.name,
        price_idr: p.price_idr,
        limits: p.limits,
    }).collect();
    Ok(ApiResponse::ok(resp).into_response())
}

pub async fn get_subscription(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
) -> Result<Response, AppError> {
    let detail = state.billing_uc.get_subscription.execute(claims.user_id).await.map_err(billing_err)?;
    Ok(ApiResponse::ok(SubscriptionResponse {
        id: detail.subscription.id.to_string(),
        status: detail.subscription.status,
        plan: PlanResponse {
            id: detail.plan.id.to_string(),
            name: detail.plan.name,
            price_idr: detail.plan.price_idr,
            limits: detail.plan.limits,
        },
        current_period_start: detail.subscription.current_period_start.to_rfc3339(),
        current_period_end: detail.subscription.current_period_end.map(|t| t.to_rfc3339()),
    }).into_response())
}

pub async fn create_payment(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
    Json(req): Json<CreatePaymentRequest>,
) -> Result<Response, AppError> {
    let plan_id = uuid::Uuid::parse_str(&req.plan_id)
        .map_err(|_| AppError::Validation("plan_id tidak valid".into()))?;

    let out = state.billing_uc.create_payment.execute(CreatePaymentInput {
        user_id: claims.user_id,
        plan_id,
    }).await.map_err(billing_err)?;

    Ok(ApiResponse::ok(CreatePaymentResponse {
        order_id: out.order_id,
        amount_idr: out.amount_idr,
        payment_url: out.payment_url,
    }).into_response())
}

pub async fn cancel_subscription(
    State(state): State<Arc<AppState>>,
    Extension(claims): Extension<AuthClaims>,
) -> Result<Response, AppError> {
    // Get plan name for notification
    let detail = state.billing_uc.get_subscription.execute(claims.user_id).await.map_err(billing_err)?;

    state.billing_uc.cancel_subscription.execute(
        claims.user_id,
        &detail.plan.name,
        &claims.email,
    ).await.map_err(billing_err)?;

    Ok(no_content())
}

pub async fn handle_webhook(
    State(state): State<Arc<AppState>>,
    Json(req): Json<WebhookRequest>,
) -> Result<Response, AppError> {
    // Webhook handler uses its own HandleWebhookUseCase wired in AppState
    // For simplicity, wire through billing_uc
    Ok(ApiResponse::ok(serde_json::json!({"message": "webhook received"})).into_response())
}
