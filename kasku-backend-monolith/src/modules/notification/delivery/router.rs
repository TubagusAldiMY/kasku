use std::sync::Arc;
use axum::{middleware, routing::get, Router};

use crate::app_state::AppState;
use crate::shared::middleware::auth::auth_middleware;
use super::handler;

pub fn notification_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/preferences", get(handler::get_preference).put(handler::update_preference))
        .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
}
