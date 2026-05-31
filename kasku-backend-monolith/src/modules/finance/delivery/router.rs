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

pub fn finance_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/", get(handler::list_accounts).post(handler::create_account))
        .route(
            "/:id",
            put(handler::update_account).delete(handler::delete_account),
        )
        .route("/:id/history", get(handler::get_balance_history))
        .route_layer(middleware::from_fn(tenant_inject_middleware))
        .route_layer(middleware::from_fn_with_state(
            state.clone(),
            tier_inject_middleware,
        ))
        .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
}
