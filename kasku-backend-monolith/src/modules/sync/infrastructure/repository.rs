use chrono::{DateTime, Utc};
use serde_json::Value as JsonValue;
use sqlx::Row;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::sync::domain::error::SyncError;
use crate::shared::tenant::validate_tenant_schema;

/// Repository for sync_log and conflict detection in tenant schemas.
#[derive(Clone)]
pub struct SyncRepository {
    pool: PgPool,
}

impl SyncRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// Check idempotency: whether sync_id has already been processed.
    pub async fn sync_id_exists(
        &self,
        tenant_schema: &str,
        sync_id: &Uuid,
    ) -> Result<bool, SyncError> {
        validate_tenant_schema(tenant_schema)
            .map_err(|_| SyncError::InvalidTenantSchema(tenant_schema.to_string()))?;

        let query = format!(
            "SELECT EXISTS(SELECT 1 FROM {}.sync_log WHERE id = $1)",
            tenant_schema
        );
        let result = sqlx::query_scalar::<_, bool>(&query)
            .bind(sync_id)
            .fetch_one(&self.pool)
            .await;

        match result {
            Ok(exists) => Ok(exists),
            Err(sqlx::Error::Database(ref db_err))
                if db_err.code().as_deref() == Some("42P01") =>
            {
                Err(SyncError::TenantNotProvisioned(tenant_schema.to_string()))
            }
            Err(e) => Err(SyncError::DatabaseError(e)),
        }
    }

    /// Get server entity for conflict detection (Server Wins).
    /// Returns (entity_data, updated_at) or None if not found.
    pub async fn get_entity(
        &self,
        tenant_schema: &str,
        entity_type: &str,
        entity_id: &Uuid,
    ) -> Result<Option<(JsonValue, DateTime<Utc>)>, SyncError> {
        validate_tenant_schema(tenant_schema)
            .map_err(|_| SyncError::InvalidTenantSchema(tenant_schema.to_string()))?;

        let table = match entity_type {
            "financial_account" => "financial_accounts",
            "transaction" => "transactions",
            "investment_asset" => "investment_assets",
            _ => return Err(SyncError::UnsupportedEntityType(entity_type.to_string())),
        };

        let query = format!(
            "SELECT to_jsonb(t.*) as data, t.updated_at FROM {}.{} t WHERE t.id = $1",
            tenant_schema, table
        );

        let row = sqlx::query(&query)
            .bind(entity_id)
            .fetch_optional(&self.pool)
            .await
            .map_err(SyncError::DatabaseError)?;

        Ok(row.map(|r| {
            let data: JsonValue = r.try_get("data").unwrap_or(JsonValue::Null);
            let updated_at: DateTime<Utc> = r.try_get("updated_at").unwrap_or_default();
            (data, updated_at)
        }))
    }

    /// Record sync operation to sync_log.
    pub async fn log_sync(
        &self,
        tenant_schema: &str,
        sync_id: &Uuid,
        operation: &str,
        entity_type: &str,
        entity_id: &Uuid,
        resolution: Option<&str>,
    ) -> Result<(), SyncError> {
        validate_tenant_schema(tenant_schema)
            .map_err(|_| SyncError::InvalidTenantSchema(tenant_schema.to_string()))?;

        let query = format!(
            "INSERT INTO {}.sync_log (id, operation, entity_type, entity_id, resolution, synced_at)
             VALUES ($1, $2, $3, $4, $5, now())",
            tenant_schema
        );
        let result = sqlx::query(&query)
            .bind(sync_id)
            .bind(operation)
            .bind(entity_type)
            .bind(entity_id)
            .bind(resolution)
            .execute(&self.pool)
            .await;

        match result {
            Ok(_) => Ok(()),
            Err(sqlx::Error::Database(ref db_err))
                if db_err.code().as_deref() == Some("42P01") =>
            {
                Err(SyncError::TenantNotProvisioned(tenant_schema.to_string()))
            }
            Err(e) => Err(SyncError::DatabaseError(e)),
        }
    }
}
