use std::sync::Arc;

use crate::modules::finance::domain::repository::FinancialAccountRepository;
use crate::modules::investment::domain::repository::InvestmentRepository;
use crate::modules::sync::infrastructure::repository::SyncRepository;
use crate::modules::transaction::domain::repository::TransactionRepository;

pub mod pull_sync;
pub mod push_sync;

pub use pull_sync::PullSyncUseCase;
pub use push_sync::PushSyncUseCase;

pub struct SyncUseCases {
    pub pull: Arc<PullSyncUseCase>,
    pub push: Arc<PushSyncUseCase>,
}

impl SyncUseCases {
    pub fn new(
        sync_repo: SyncRepository,
        finance_repo: Arc<dyn FinancialAccountRepository>,
        transaction_repo: Arc<dyn TransactionRepository>,
        investment_repo: Arc<dyn InvestmentRepository>,
    ) -> Self {
        let pull = Arc::new(PullSyncUseCase::new(
            finance_repo.clone(),
            transaction_repo.clone(),
            investment_repo.clone(),
        ));
        let push = Arc::new(PushSyncUseCase::new(
            sync_repo,
            finance_repo,
            transaction_repo,
            investment_repo,
        ));
        Self { pull, push }
    }
}
