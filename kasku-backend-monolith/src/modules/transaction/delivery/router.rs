use std::sync::Arc;
use axum::{
    middleware,
    routing::{delete, get, post, put},
    Router,
};

use crate::app_state::AppState;
use crate::shared::middleware::auth::auth_middleware;
use crate::shared::middleware::tenant_inject::tenant_inject_middleware;
use crate::shared::middleware::tier_inject::tier_inject_middleware;
use super::handler;

pub fn transaction_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/", get(handler::list_transactions).post(handler::create_transaction))
        .route("/summary", get(handler::get_summary))
        .route("/export", get(handler::export_csv))
        .route("/:id", get(handler::get_transaction).put(handler::update_transaction).delete(handler::delete_transaction))
        .route_layer(middleware::from_fn(tenant_inject_middleware))
        .route_layer(middleware::from_fn_with_state(
            state.clone(),
            tier_inject_middleware,
        ))
        .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
}

pub fn category_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/", get(handler::list_categories).post(handler::create_category))
        .route("/:id", put(handler::update_category).delete(handler::delete_category))
        .route_layer(middleware::from_fn(tenant_inject_middleware))
        .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
}

pub fn budget_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/", get(handler::list_budgets).post(handler::create_budget))
        .route("/:id", get(handler::get_budget).put(handler::update_budget).delete(handler::delete_budget))
        .route_layer(middleware::from_fn(tenant_inject_middleware))
        .route_layer(middleware::from_fn_with_state(
            state.clone(),
            tier_inject_middleware,
        ))
        .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
}
