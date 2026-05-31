use std::sync::Arc;
use std::time::Duration;
use sqlx_postgres::PgPool;
use tokio_util::sync::CancellationToken;
use tracing::{info, warn};

use crate::modules::billing::domain::repository::SubscriptionRepository;

pub struct ExpireSubscriptionsUseCase {
    sub_repo: Arc<dyn SubscriptionRepository>,
    pool: PgPool,
}

impl ExpireSubscriptionsUseCase {
    pub fn new(sub_repo: Arc<dyn SubscriptionRepository>, pool: PgPool) -> Self {
        Self { sub_repo, pool }
    }

    pub async fn expire_overdue(&self) -> anyhow::Result<u64> {
        let count = self.sub_repo.expire_overdue().await
            .map_err(|e| anyhow::anyhow!(e))?;
        Ok(count)
    }

    pub async fn run(self, cancel: CancellationToken) {
        let mut ticker = tokio::time::interval(Duration::from_secs(3600));
        loop {
            tokio::select! {
                _ = cancel.cancelled() => {
                    info!("subscription expiry checker stopping");
                    return;
                }
                _ = ticker.tick() => {
                    match self.expire_overdue().await {
                        Ok(n) => if n > 0 { info!(count = n, "expired subscriptions processed") },
                        Err(e) => warn!(error = %e, "subscription expiry check failed"),
                    }
                }
            }
        }
    }
}
