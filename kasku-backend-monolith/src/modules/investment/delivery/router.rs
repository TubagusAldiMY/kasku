use std::sync::Arc;
use axum::{middleware, routing::{delete, get, post, put}, Router};

use crate::app_state::AppState;
use crate::shared::middleware::auth::auth_middleware;
use crate::shared::middleware::tenant_inject::tenant_inject_middleware;
use super::handler;

pub fn investment_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/", get(handler::list_assets).post(handler::create_asset))
        .route("/:id", get(handler::get_asset).put(handler::update_asset).delete(handler::delete_asset))
        .route("/:id/transactions", post(handler::record_unit_transaction))
        .route("/:id/history", get(handler::get_unit_history))
        .route_layer(middleware::from_fn(tenant_inject_middleware))
        .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
}
