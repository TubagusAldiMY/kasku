use chrono::Utc;
use tracing::{info, warn};

use crate::domain::entity::{PushResponse, SyncOperation, SyncResult, SyncStatus};
use crate::domain::error::DomainError;
use crate::infrastructure::repository::SyncRepository;
use crate::infrastructure::tenant::{validate_tenant_schema, verify_tenant_ownership};

/// Use case: Process a batch of offline sync operations.
///
/// For each operation:
/// 1. Check sync_id for idempotency (skip if already processed)
/// 2. Check for conflicts: if server entity updated_at > client_timestamp → SERVER WINS
/// 3. Otherwise, apply the operation
/// 4. Log to sync_log
pub struct PushSyncUseCase {
    repo: SyncRepository,
}

impl PushSyncUseCase {
    pub fn new(repo: SyncRepository) -> Self {
        Self { repo }
    }

    pub async fn execute(
        &self,
        user_id: &str,
        tenant_schema: &str,
        operations: Vec<SyncOperation>,
    ) -> Result<PushResponse, DomainError> {
        // Security: validate tenant schema and ownership
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
            if self
                .repo
                .sync_id_exists(tenant_schema, &op.sync_id)
                .await?
            {
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

            // Step 2: Conflict detection (for update/delete operations)
            if op.operation == "update" || op.operation == "delete" {
                if let Some((server_data, server_updated_at)) = self
                    .repo
                    .get_entity(tenant_schema, &op.entity_type, &op.entity_id)
                    .await?
                {
                    if server_updated_at > op.client_timestamp {
                        // SERVER WINS — return server data to client
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

            // Step 3: Apply operation (log only — actual entity mutation
            // is done by the owning service via API calls)
            let log_operation = match op.operation.as_str() {
                "create" | "update" | "delete" => "PUSH",
                _ => {
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

            // Step 4: Log to sync_log
            self.repo
                .log_sync(
                    tenant_schema,
                    &op.sync_id,
                    log_operation,
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
}
