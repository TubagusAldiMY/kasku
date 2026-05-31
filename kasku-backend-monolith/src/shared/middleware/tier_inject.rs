use axum::{extract::{Request, State}, middleware::Next, response::Response};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

use crate::app_error::AppError;
use crate::app_state::AppState;
use crate::modules::billing::usecase::GetTierLimitsUseCase;
use crate::shared::middleware::auth::AuthClaims;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TierLimits {
    pub max_transactions_per_month: i64,
    pub max_financial_accounts: i64,
    pub max_investment_instruments: i64,
    pub history_retention_months: i64,
    pub email_notifications_enabled: bool,
    pub export_csv_enabled: bool,
}

impl TierLimits {
    pub fn free() -> Self {
        Self {
            max_transactions_per_month: 50,
            max_financial_accounts: 3,
            max_investment_instruments: 0,
            history_retention_months: 3,
            email_notifications_enabled: false,
            export_csv_enabled: false,
        }
    }

    pub fn is_unlimited(&self, field: i64) -> bool {
        field == -1
    }
}

pub async fn tier_inject_middleware(
    State(state): State<Arc<AppState>>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    let user_id = {
        let claims = req.extensions().get::<AuthClaims>().cloned()
            .ok_or_else(|| AppError::Unauthorized("auth claims not found".into()))?;
        claims.user_id
    };

    let limits = state
        .billing_uc
        .get_tier_limits(user_id)
        .await
        .unwrap_or_else(|_| TierLimits::free());

    req.extensions_mut().insert(limits);
    Ok(next.run(req).await)
}
