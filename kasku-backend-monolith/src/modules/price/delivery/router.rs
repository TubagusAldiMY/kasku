use std::sync::Arc;
use axum::{routing::get, Router};

use crate::app_state::AppState;
use super::handler::get_price;

pub fn price_router(state: Arc<AppState>) -> Router<Arc<AppState>> {
    Router::new()
        .route("/:symbol", get(get_price))
        .with_state(state)
}
