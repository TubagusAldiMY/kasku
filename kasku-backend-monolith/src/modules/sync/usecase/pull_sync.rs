use std::sync::Arc;
use chrono::{DateTime, Utc};
use tracing::error;

use crate::modules::finance::domain::repository::FinancialAccountRepository;
use crate::modules::investment::domain::repository::InvestmentRepository;
use crate::modules::sync::domain::entity::{EntityChange, PullResponse};
use crate::modules::sync::domain::error::SyncError;
use crate::modules::transaction::domain::repository::TransactionRepository;
use crate::shared::tenant::{validate_tenant_schema, user_id_to_schema};

/// Pull sync use case.
///
/// Fan-out to 3 repositories in parallel via tokio::join!, aggregate results,
/// and return all changes since `since` sorted by updated_at.
pub struct PullSyncUseCase {
    finance_repo: Arc<dyn FinancialAccountRepository>,
    transaction_repo: Arc<dyn TransactionRepository>,
    investment_repo: Arc<dyn InvestmentRepository>,
}

impl PullSyncUseCase {
    pub fn new(
        finance_repo: Arc<dyn FinancialAccountRepository>,
        transaction_repo: Arc<dyn TransactionRepository>,
        investment_repo: Arc<dyn InvestmentRepository>,
    ) -> Self {
        Self {
            finance_repo,
            transaction_repo,
            investment_repo,
        }
    }

    pub async fn execute(
        &self,
        user_id: &str,
        tenant_schema: &str,
        since: DateTime<Utc>,
    ) -> Result<PullResponse, SyncError> {
        validate_tenant_schema(tenant_schema)
            .map_err(|_| SyncError::InvalidTenantSchema(tenant_schema.to_string()))?;
        verify_tenant_ownership(user_id, tenant_schema)?;

        // Fan-out parallel to all repos
        let (finance_result, tx_result, invest_result) = tokio::join!(
            self.finance_repo.list_since(tenant_schema, since),
            self.transaction_repo.list_since(tenant_schema, since),
            self.investment_repo.list_since(tenant_schema, since),
        );

        let mut changes: Vec<EntityChange> = Vec::new();

        match finance_result {
            Ok(accounts) => {
                for account in accounts {
                    match serde_json::to_value(&account) {
                        Ok(data) => changes.push(EntityChange {
                            entity_type: "financial_account".to_string(),
                            entity_id: account.id,
                            operation: "upsert".to_string(),
                            data,
                            updated_at: account.updated_at,
                        }),
                        Err(e) => {
                            error!(error = %e, "gagal serialize financial_account untuk pull sync");
                        }
                    }
                }
            }
            Err(e) => {
                error!(error = %e, "gagal ambil financial_accounts untuk pull sync");
            }
        }

        match tx_result {
            Ok(transactions) => {
                for tx in transactions {
                    match serde_json::to_value(&tx) {
                        Ok(data) => changes.push(EntityChange {
                            entity_type: "transaction".to_string(),
                            entity_id: tx.id,
                            operation: "upsert".to_string(),
                            data,
                            updated_at: tx.updated_at,
                        }),
                        Err(e) => {
                            error!(error = %e, "gagal serialize transaction untuk pull sync");
                        }
                    }
                }
            }
            Err(e) => {
                error!(error = %e, "gagal ambil transactions untuk pull sync");
            }
        }

        match invest_result {
            Ok(assets) => {
                for asset in assets {
                    match serde_json::to_value(&asset) {
                        Ok(data) => changes.push(EntityChange {
                            entity_type: "investment_asset".to_string(),
                            entity_id: asset.id,
                            operation: "upsert".to_string(),
                            data,
                            updated_at: asset.updated_at,
                        }),
                        Err(e) => {
                            error!(error = %e, "gagal serialize investment_asset untuk pull sync");
                        }
                    }
                }
            }
            Err(e) => {
                error!(error = %e, "gagal ambil investment_assets untuk pull sync");
            }
        }

        changes.sort_by_key(|c| c.updated_at);

        Ok(PullResponse {
            changes,
            server_timestamp: Utc::now(),
        })
    }
}

/// Verify that the tenant schema matches the user ID.
fn verify_tenant_ownership(user_id: &str, tenant_schema: &str) -> Result<(), SyncError> {
    let expected = user_id_to_schema(user_id);
    if expected != tenant_schema {
        return Err(SyncError::TenantMismatch {
            expected,
            actual: tenant_schema.to_string(),
        });
    }
    Ok(())
}
