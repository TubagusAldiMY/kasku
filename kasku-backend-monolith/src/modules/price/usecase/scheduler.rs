use std::sync::Arc;
use std::time::Duration;
use tokio_util::sync::CancellationToken;

use crate::modules::price::usecase::get_price::GetPriceUseCase;

pub struct PriceScheduler {
    pub uc: Arc<GetPriceUseCase>,
    pub symbols: Vec<String>,
    pub interval_seconds: u64,
}

impl PriceScheduler {
    pub async fn run(self, cancel: CancellationToken) {
        let mut interval = tokio::time::interval(Duration::from_secs(self.interval_seconds));
        loop {
            tokio::select! {
                _ = cancel.cancelled() => break,
                _ = interval.tick() => {
                    for symbol in &self.symbols {
                        if let Err(e) = self.uc.execute(symbol, "METALS_LIVE").await {
                            tracing::warn!(symbol=%symbol, error=%e, "price scheduler fetch failed");
                        }
                    }
                }
            }
        }
    }
}
