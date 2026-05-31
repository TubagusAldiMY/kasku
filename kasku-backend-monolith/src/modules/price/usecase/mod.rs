pub mod fetch_external;
pub mod get_price;
pub mod scheduler;

pub use get_price::GetPriceUseCase;
pub use scheduler::PriceScheduler;

use std::sync::Arc;

/// Aggregate of price use cases exposed on AppState.
pub struct PriceUseCases {
    pub get_price: Arc<GetPriceUseCase>,
    pub scheduler: Option<PriceScheduler>,
}

impl PriceUseCases {
    pub fn new(get_price: Arc<GetPriceUseCase>) -> Self {
        Self {
            get_price,
            scheduler: None,
        }
    }

    /// Convenience: execute get_price use case.
    pub async fn execute(
        &self,
        symbol: &str,
        source_hint: &str,
    ) -> Result<crate::modules::price::domain::entity::PriceResult, crate::modules::price::domain::error::PriceError> {
        self.get_price.execute(symbol, source_hint).await
    }
}
