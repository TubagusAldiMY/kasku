use chrono::{DateTime, Utc};

use crate::domain::entity::PullResponse;
use crate::domain::error::DomainError;
use crate::infrastructure::repository::SyncRepository;
use crate::infrastructure::tenant::{validate_tenant_schema, verify_tenant_ownership};

/// Use case: Pull all changes since a given timestamp.
///
/// Returns delta changes across financial_accounts, transactions, and investment_assets.
pub struct PullSyncUseCase {
    repo: SyncRepository,
}

impl PullSyncUseCase {
    pub fn new(repo: SyncRepository) -> Self {
        Self { repo }
    }

    pub async fn execute(
        &self,
        user_id: &str,
        tenant_schema: &str,
        since: DateTime<Utc>,
    ) -> Result<PullResponse, DomainError> {
        // Security: validate tenant schema and ownership
        validate_tenant_schema(tenant_schema)?;
        verify_tenant_ownership(user_id, tenant_schema)?;

        let changes = self.repo.get_changes_since(tenant_schema, since).await?;

        Ok(PullResponse {
            changes,
            server_timestamp: Utc::now(),
        })
    }
}
