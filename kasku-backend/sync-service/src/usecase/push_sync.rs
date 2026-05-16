use chrono::Utc;
use tracing::{info, warn};

use crate::domain::entity::{PushResponse, SyncOperation, SyncResult, SyncStatus};
use crate::domain::error::DomainError;
use crate::infrastructure::grpc::SyncGrpcClients;
use crate::infrastructure::repository::SyncRepository;
use crate::infrastructure::tenant::{validate_tenant_schema, verify_tenant_ownership};
use crate::proto::{
    common::SyncUpsertItem,
    finance::UpsertFinancialAccountsRequest,
    investment::UpsertInvestmentAssetsRequest,
    transaction::UpsertTransactionsRequest,
};

/// Push sync use case.
///
/// Alur:
/// 1. Idempotency check via sync_log (direct DB)
/// 2. Conflict detection: baca updated_at entity dari DB (direct DB, Server Wins)
/// 3. Apply operation: panggil owning service via gRPC
/// 4. Log ke sync_log (direct DB)
pub struct PushSyncUseCase {
    repo: SyncRepository,
    grpc: SyncGrpcClients,
}

impl PushSyncUseCase {
    pub fn new(repo: SyncRepository, grpc: SyncGrpcClients) -> Self {
        Self { repo, grpc }
    }

    pub async fn execute(
        &self,
        user_id: &str,
        tenant_schema: &str,
        operations: Vec<SyncOperation>,
    ) -> Result<PushResponse, DomainError> {
        validate_tenant_schema(tenant_schema)?;
        verify_tenant_ownership(user_id, tenant_schema)?;

        let mut results = Vec::with_capacity(operations.len());
        let mut processed = 0usize;
        let mut conflicts = 0usize;
        let mut skipped = 0usize;

        for op in &operations {
            // Validate entity type
            if !["financial_account", "transaction", "investment_asset"]
                .contains(&op.entity_type.as_str())
            {
                results.push(SyncResult {
                    sync_id: op.sync_id,
                    entity_type: op.entity_type.clone(),
                    entity_id: op.entity_id,
                    status: SyncStatus::Error,
                    server_data: None,
                });
                continue;
            }

            // Step 1: Idempotency check
            if self.repo.sync_id_exists(tenant_schema, &op.sync_id).await? {
                skipped += 1;
                results.push(SyncResult {
                    sync_id: op.sync_id,
                    entity_type: op.entity_type.clone(),
                    entity_id: op.entity_id,
                    status: SyncStatus::Skipped,
                    server_data: None,
                });
                continue;
            }

            // Step 2: Conflict detection (update/delete only)
            if op.operation == "update" || op.operation == "delete" {
                if let Some((server_data, server_updated_at)) = self
                    .repo
                    .get_entity(tenant_schema, &op.entity_type, &op.entity_id)
                    .await?
                {
                    if server_updated_at > op.client_timestamp {
                        conflicts += 1;
                        self.repo
                            .log_sync(
                                tenant_schema,
                                &op.sync_id,
                                "CONFLICT_RESOLVED",
                                &op.entity_type,
                                &op.entity_id,
                                Some("SERVER_WINS"),
                            )
                            .await?;

                        warn!(
                            sync_id = %op.sync_id,
                            entity_type = op.entity_type.as_str(),
                            entity_id = %op.entity_id,
                            "konflik sync terdeteksi, server menang"
                        );

                        results.push(SyncResult {
                            sync_id: op.sync_id,
                            entity_type: op.entity_type.clone(),
                            entity_id: op.entity_id,
                            status: SyncStatus::Conflict,
                            server_data: Some(server_data),
                        });
                        continue;
                    }
                }
            }

            // Step 3: Apply via gRPC ke owning service
            let payload_bytes = match serde_json::to_vec(&op.payload) {
                Ok(b) => b,
                Err(e) => {
                    warn!(sync_id = %op.sync_id, error = %e, "gagal serialize payload");
                    results.push(SyncResult {
                        sync_id: op.sync_id,
                        entity_type: op.entity_type.clone(),
                        entity_id: op.entity_id,
                        status: SyncStatus::Error,
                        server_data: None,
                    });
                    continue;
                }
            };

            let item = SyncUpsertItem {
                sync_id: op.sync_id.to_string(),
                entity_id: op.entity_id.to_string(),
                operation: op.operation.clone(),
                payload: payload_bytes,
                client_ts_ms: op.client_timestamp.timestamp_millis(),
            };

            if let Err(err) = self
                .apply_via_grpc(tenant_schema, user_id, &op.entity_type, item)
                .await
            {
                warn!(
                    sync_id = %op.sync_id,
                    entity_type = op.entity_type.as_str(),
                    entity_id = %op.entity_id,
                    error = %err,
                    "gagal apply operasi sync via gRPC"
                );
                results.push(SyncResult {
                    sync_id: op.sync_id,
                    entity_type: op.entity_type.clone(),
                    entity_id: op.entity_id,
                    status: SyncStatus::Error,
                    server_data: None,
                });
                continue;
            }

            // Step 4: Log ke sync_log
            self.repo
                .log_sync(
                    tenant_schema,
                    &op.sync_id,
                    "PUSH",
                    &op.entity_type,
                    &op.entity_id,
                    Some("NO_CONFLICT"),
                )
                .await?;

            processed += 1;
            results.push(SyncResult {
                sync_id: op.sync_id,
                entity_type: op.entity_type.clone(),
                entity_id: op.entity_id,
                status: SyncStatus::Applied,
                server_data: None,
            });
        }

        info!(
            processed = processed,
            conflicts = conflicts,
            skipped = skipped,
            total = operations.len(),
            "batch sync selesai"
        );

        Ok(PushResponse {
            processed,
            conflicts,
            skipped,
            results,
            server_timestamp: Utc::now(),
        })
    }

    async fn apply_via_grpc(
        &self,
        tenant_schema: &str,
        user_id: &str,
        entity_type: &str,
        item: SyncUpsertItem,
    ) -> Result<(), DomainError> {
        let schema = tenant_schema.to_string();
        let uid = user_id.to_string();

        match entity_type {
            "financial_account" => {
                let resp = self
                    .grpc
                    .finance
                    .upsert_financial_accounts(UpsertFinancialAccountsRequest {
                        tenant_schema: schema,
                        user_id: uid,
                        items: vec![item],
                    })
                    .await
                    .map_err(|e| DomainError::GrpcError(e.to_string()))?;

                if resp.results.first().map(|r| r.status.as_str()) != Some("applied") {
                    return Err(DomainError::GrpcError(
                        "finance-service menolak operasi sync".into(),
                    ));
                }
            }
            "transaction" => {
                let resp = self
                    .grpc
                    .transaction
                    .upsert_transactions(UpsertTransactionsRequest {
                        tenant_schema: schema,
                        user_id: uid,
                        items: vec![item],
                    })
                    .await
                    .map_err(|e| DomainError::GrpcError(e.to_string()))?;

                if resp.results.first().map(|r| r.status.as_str()) != Some("applied") {
                    return Err(DomainError::GrpcError(
                        "transaction-service menolak operasi sync".into(),
                    ));
                }
            }
            "investment_asset" => {
                let resp = self
                    .grpc
                    .investment
                    .upsert_investment_assets(UpsertInvestmentAssetsRequest {
                        tenant_schema: schema,
                        user_id: uid,
                        items: vec![item],
                    })
                    .await
                    .map_err(|e| DomainError::GrpcError(e.to_string()))?;

                if resp.results.first().map(|r| r.status.as_str()) != Some("applied") {
                    return Err(DomainError::GrpcError(
                        "investment-service menolak operasi sync".into(),
                    ));
                }
            }
            other => return Err(DomainError::UnsupportedEntityType(other.to_string())),
        }
        Ok(())
    }
}
