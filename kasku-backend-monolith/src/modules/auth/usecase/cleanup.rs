use std::sync::Arc;
use std::time::Duration;
use chrono::Utc;
use tokio_util::sync::CancellationToken;
use tracing::{info, warn};

use crate::modules::auth::domain::repository::RefreshTokenRepository;

pub struct AuthCleanupTask {
    refresh_token_repo: Arc<dyn RefreshTokenRepository>,
    interval: Duration,
}

impl AuthCleanupTask {
    pub fn new(refresh_token_repo: Arc<dyn RefreshTokenRepository>, interval: Duration) -> Self {
        Self { refresh_token_repo, interval }
    }

    pub async fn run(self, cancel: CancellationToken) {
        let mut ticker = tokio::time::interval(self.interval);
        loop {
            tokio::select! {
                _ = cancel.cancelled() => {
                    info!("auth cleanup task stopping");
                    return;
                }
                _ = ticker.tick() => {
                    let cutoff = Utc::now();
                    match self.refresh_token_repo.delete_expired(cutoff).await {
                        Ok(n) => info!(count = n, "expired refresh tokens cleaned up"),
                        Err(e) => warn!(error = %e, "auth cleanup failed"),
                    }
                }
            }
        }
    }
}
