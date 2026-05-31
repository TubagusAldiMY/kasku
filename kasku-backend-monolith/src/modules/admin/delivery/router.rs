use std::sync::Arc;
use axum::{
    middleware,
    routing::{get, post},
    Router,
};

use crate::app_state::AppState;
use crate::shared::middleware::admin_auth::admin_auth_middleware;

use super::handler;

pub fn admin_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    let protected = Router::new()
        .route("/me", get(handler::get_current))
        .route("/dashboard", get(handler::dashboard_stats))
        .route("/users", get(handler::list_users))
        .route("/users/:id", get(handler::get_user))
        .route("/users/:id/suspend", post(handler::suspend_user))
        .route("/users/:id/activate", post(handler::activate_user))
        .route("/audit-log", get(handler::list_audit_log))
        .route_layer(middleware::from_fn_with_state(state.clone(), admin_auth_middleware));

    Router::new()
        .route("/auth/login", post(handler::login))
        .merge(protected)
}
