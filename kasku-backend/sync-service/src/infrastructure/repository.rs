use chrono::{DateTime, Utc};
use serde_json::Value as JsonValue;
use sqlx::PgPool;
use uuid::Uuid;

use crate::domain::error::DomainError;
use crate::infrastructure::tenant::validate_tenant_schema;

/// Repository untuk sync_log dan conflict detection di kasku_finance.
/// apply_operation dan get_changes_since dipindahkan ke owning service via gRPC.
#[derive(Clone)]
pub struct SyncRepository {
    pool: PgPool,
}

impl SyncRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// Check idempotency: apakah sync_id sudah pernah diproses.
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
        let result = sqlx::query_scalar::<_, bool>(&query)
            .bind(sync_id)
            .fetch_one(&self.pool)
            .await;

        match result {
            Ok(exists) => Ok(exists),
            Err(sqlx::Error::Database(ref db_err))
                if db_err.code().as_deref() == Some("42P01") =>
            {
                // PostgreSQL error 42P01: "relation does not exist"
                // sync_log belum dibuat — tenant belum di-provision oleh finance-service.
                Err(DomainError::TenantNotProvisioned(tenant_schema.to_string()))
            }
            Err(e) => Err(DomainError::DatabaseError(e)),
        }
    }

    /// Ambil versi server entity untuk conflict detection (Server Wins).
    /// Mengembalikan (entity_data, updated_at) atau None jika tidak ada.
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
            .await?;

        Ok(row)
    }

    /// Catat operasi sync ke sync_log.
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
                Err(DomainError::TenantNotProvisioned(tenant_schema.to_string()))
            }
            Err(e) => Err(DomainError::DatabaseError(e)),
        }
    }
}
