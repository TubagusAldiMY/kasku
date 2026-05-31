use std::sync::Arc;
use axum::{
    middleware,
    routing::{get, post, put},
    Router,
};

use crate::app_state::AppState;
use crate::shared::middleware::auth::auth_middleware;
use super::handler;

pub fn auth_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/register", post(handler::register))
        .route("/login", post(handler::login))
        .route("/refresh", post(handler::refresh))
        .route("/verify-email", get(handler::verify_email))
        .route("/resend-verification", post(handler::resend_verification))
        .route("/forgot-password", post(handler::forgot_password))
        .route("/reset-password", post(handler::reset_password))
        // Protected routes
        .route(
            "/logout",
            post(handler::logout)
                .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware)),
        )
        .route(
            "/change-password",
            put(handler::change_password)
                .route_layer(middleware::from_fn_with_state(state.clone(), auth_middleware)),
        )
}
