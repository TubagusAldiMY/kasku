use std::sync::Arc;
use axum::{middleware, routing::get, Router};

use crate::app_state::AppState;
use crate::shared::middleware::auth::auth_middleware;
use super::handler;

pub fn user_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route(
            "/profile",
            get(handler::get_profile)
                .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware)),
        )
}
