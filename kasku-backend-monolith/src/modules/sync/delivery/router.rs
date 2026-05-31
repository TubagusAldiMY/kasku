use std::sync::Arc;
use axum::{middleware, routing::{get, post}, Router};

use crate::app_state::AppState;
use crate::shared::middleware::auth::auth_middleware;
use crate::shared::middleware::tenant_inject::tenant_inject_middleware;
use super::handler;

pub fn sync_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/push", post(handler::push_sync))
        .route("/pull", get(handler::pull_sync))
        .route_layer(middleware::from_fn(tenant_inject_middleware))
        .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
}
