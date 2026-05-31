use std::sync::Arc;
use axum::{
    middleware,
    routing::{delete, get, post},
    Router,
};

use crate::app_state::AppState;
use crate::shared::middleware::auth::auth_middleware;
use super::handler;

pub fn billing_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/plans", get(handler::list_plans))
        .route("/webhook", post(handler::handle_webhook))
        .route(
            "/subscription",
            get(handler::get_subscription)
                .delete(handler::cancel_subscription)
                .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware)),
        )
        .route(
            "/payment",
            post(handler::create_payment)
                .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware)),
        )
}
