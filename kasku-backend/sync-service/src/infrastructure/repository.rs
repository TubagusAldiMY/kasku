use chrono::{DateTime, Utc};
use serde_json::Value as JsonValue;
use sqlx::PgPool;
use uuid::Uuid;

use crate::domain::entity::EntityChange;
use crate::domain::error::DomainError;
use crate::infrastructure::tenant::validate_tenant_schema;

/// Repository for sync operations on tenant schemas.
#[derive(Clone)]
pub struct SyncRepository {
    pool: PgPool,
}

impl SyncRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// Check if a sync_id has already been processed (idempotency check).
    pub async fn sync_id_exists(
        &self,
        tenant_schema: &str,
        sync_id: &Uuid,
    ) -> Result<bool, DomainError> {
        validate_tenant_schema(tenant_schema)?;
        let query = format!(
            "SELECT EXISTS(SELECT 1 FROM {}.sync_log WHERE id = $1)",
            tenant_schema
        );
        let exists: bool = sqlx::query_scalar(&query)
            .bind(sync_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;
        Ok(exists)
    }

    /// Get the server's current version of an entity (for conflict detection).
    /// Returns (entity_data, updated_at).
    pub async fn get_entity(
        &self,
        tenant_schema: &str,
        entity_type: &str,
        entity_id: &Uuid,
    ) -> Result<Option<(JsonValue, DateTime<Utc>)>, DomainError> {
        validate_tenant_schema(tenant_schema)?;

        let table = match entity_type {
            "financial_account" => "financial_accounts",
            "transaction" => "transactions",
            "investment_asset" => "investment_assets",
            _ => return Err(DomainError::UnsupportedEntityType(entity_type.to_string())),
        };

        let query = format!(
            "SELECT to_jsonb(t.*) as data, t.updated_at FROM {}.{} t WHERE t.id = $1",
            tenant_schema, table
        );

        let row: Option<(JsonValue, DateTime<Utc>)> = sqlx::query_as(&query)
            .bind(entity_id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        Ok(row)
    }

    /// Log a sync operation to the sync_log table.
    pub async fn log_sync(
        &self,
        tenant_schema: &str,
        sync_id: &Uuid,
        operation: &str,
        entity_type: &str,
        entity_id: &Uuid,
        resolution: Option<&str>,
    ) -> Result<(), DomainError> {
        validate_tenant_schema(tenant_schema)?;
        let query = format!(
            "INSERT INTO {}.sync_log (id, operation, entity_type, entity_id, resolution, synced_at)
             VALUES ($1, $2, $3, $4, $5, now())",
            tenant_schema
        );
        sqlx::query(&query)
            .bind(sync_id)
            .bind(operation)
            .bind(entity_type)
            .bind(entity_id)
            .bind(resolution)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;
        Ok(())
    }

    /// Get all changes since a given timestamp for pull sync.
    pub async fn get_changes_since(
        &self,
        tenant_schema: &str,
        since: DateTime<Utc>,
    ) -> Result<Vec<EntityChange>, DomainError> {
        validate_tenant_schema(tenant_schema)?;
        let mut changes = Vec::new();

        // Query each entity table for changes
        let tables = [
            ("financial_accounts", "financial_account"),
            ("transactions", "transaction"),
            ("investment_assets", "investment_asset"),
        ];

        for (table, entity_type) in &tables {
            let query = format!(
                "SELECT id, to_jsonb(t.*) as data, updated_at,
                        CASE WHEN is_deleted THEN 'delete' ELSE 'upsert' END as operation
                 FROM {}.{} t
                 WHERE updated_at > $1
                 ORDER BY updated_at ASC",
                tenant_schema, table
            );

            let rows: Vec<(Uuid, JsonValue, DateTime<Utc>, String)> = sqlx::query_as(&query)
                .bind(since)
                .fetch_all(&self.pool)
                .await
                .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

            for (id, data, updated_at, operation) in rows {
                changes.push(EntityChange {
                    entity_type: entity_type.to_string(),
                    entity_id: id,
                    operation,
                    data,
                    updated_at,
                });
            }
        }

        // Sort all changes by updated_at across entity types
        changes.sort_by_key(|c| c.updated_at);
        Ok(changes)
    }
}
