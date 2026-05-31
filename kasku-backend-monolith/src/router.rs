use std::sync::Arc;
use axum::{
    Json,
    Router,
    routing::get,
};
use serde_json::json;
use tower_http::cors::{Any, CorsLayer};
use tower_http::trace::TraceLayer;

use crate::app_state::AppState;
use crate::modules::admin::delivery::router::admin_router;
use crate::modules::auth::delivery::router::auth_router;
use crate::modules::billing::delivery::router::billing_router;
use crate::modules::finance::delivery::router::finance_router;
use crate::modules::investment::delivery::router::investment_router;
use crate::modules::notification::delivery::router::notification_router;
use crate::modules::price::delivery::router::price_router;
use crate::modules::sync::delivery::router::sync_router;
use crate::modules::transaction::delivery::router::{budget_router, category_router, transaction_router};
use crate::modules::user::delivery::router::user_router;

pub fn create_router(state: Arc<AppState>, cors_origins: Vec<String>) -> Router {
    let cors = build_cors(&cors_origins);

    let api = Router::new()
        .nest("/auth", auth_router(state.clone()))
        .nest("/billing", billing_router(state.clone()))
        .nest("/user", user_router(state.clone()))
        .nest("/finance/accounts", finance_router(state.clone()))
        .nest("/transactions", transaction_router(state.clone()))
        .nest("/categories", category_router(state.clone()))
        .nest("/budgets", budget_router(state.clone()))
        .nest("/investments", investment_router(state.clone()))
        .nest("/prices", price_router(state.clone()))
        .nest("/sync", sync_router(state.clone()))
        .nest("/notification", notification_router(state.clone()))
        .nest("/admin", admin_router(state.clone()));

    Router::new()
        .route("/health", get(health_handler))
        .nest("/v1", api)
        .layer(cors)
        .layer(TraceLayer::new_for_http())
        .with_state(state)
}

async fn health_handler() -> Json<serde_json::Value> {
    Json(json!({ "success": true, "data": { "status": "ok" } }))
}

fn build_cors(origins: &[String]) -> CorsLayer {
    if origins.is_empty() || origins.iter().any(|o| o == "*") {
        return CorsLayer::new()
            .allow_origin(Any)
            .allow_methods(Any)
            .allow_headers(Any);
    }

    use axum::http::HeaderValue;
    let allowed: Vec<HeaderValue> = origins
        .iter()
        .filter_map(|o| o.parse().ok())
        .collect();

    CorsLayer::new()
        .allow_origin(allowed)
        .allow_methods(Any)
        .allow_headers(Any)
}
