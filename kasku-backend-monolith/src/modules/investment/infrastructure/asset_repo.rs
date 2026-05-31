use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::investment::domain::entity::{InvestmentAsset, UnitHistory};
use crate::modules::investment::domain::error::InvestmentError;
use crate::modules::investment::domain::repository::InvestmentRepository;
use crate::shared::tenant::validate_tenant_schema;

pub struct PostgresInvestmentRepository {
    pool: PgPool,
}

impl PostgresInvestmentRepository {
    pub fn new(pool: PgPool) -> Self { Self { pool } }
}

#[async_trait]
impl InvestmentRepository for PostgresInvestmentRepository {
    async fn list(&self, schema: &str) -> Result<Vec<InvestmentAsset>, InvestmentError> {
        let schema = validate_tenant_schema(schema)
            .map_err(|_| InvestmentError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, name, asset_type, symbol, quantity, avg_buy_price, currency, \
             is_deleted, deleted_at, sort_order, created_at, updated_at \
             FROM {schema}.investment_assets WHERE is_deleted=false ORDER BY sort_order ASC, created_at ASC"
        );
        Ok(sqlx::query_as(&q).fetch_all(&self.pool).await?)
    }

    async fn find_by_id(&self, schema: &str, id: Uuid) -> Result<Option<InvestmentAsset>, InvestmentError> {
        let schema = validate_tenant_schema(schema)
            .map_err(|_| InvestmentError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, name, asset_type, symbol, quantity, avg_buy_price, currency, \
             is_deleted, deleted_at, sort_order, created_at, updated_at \
             FROM {schema}.investment_assets WHERE id=$1 AND is_deleted=false LIMIT 1"
        );
        Ok(sqlx::query_as(&q).bind(id).fetch_optional(&self.pool).await?)
    }

    async fn create(&self, schema: &str, asset: &InvestmentAsset) -> Result<(), InvestmentError> {
        let schema = validate_tenant_schema(schema)
            .map_err(|_| InvestmentError::TenantNotProvisioned)?;
        let q = format!(
            "INSERT INTO {schema}.investment_assets \
             (id, name, asset_type, symbol, quantity, avg_buy_price, currency, is_deleted, sort_order, created_at, updated_at) \
             VALUES ($1,$2,$3,$4,$5,$6,$7,false,$8,$9,$9)"
        );
        sqlx::query(&q)
            .bind(asset.id)
            .bind(&asset.name)
            .bind(asset.asset_type.to_string())
            .bind(&asset.symbol)
            .bind(asset.quantity)
            .bind(asset.avg_buy_price)
            .bind(&asset.currency)
            .bind(asset.sort_order)
            .bind(asset.created_at)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn update(&self, schema: &str, asset: &InvestmentAsset) -> Result<(), InvestmentError> {
        let schema = validate_tenant_schema(schema)
            .map_err(|_| InvestmentError::TenantNotProvisioned)?;
        let q = format!(
            "UPDATE {schema}.investment_assets \
             SET name=$2, asset_type=$3, symbol=$4, quantity=$5, avg_buy_price=$6, currency=$7, sort_order=$8, updated_at=now() \
             WHERE id=$1 AND is_deleted=false"
        );
        sqlx::query(&q)
            .bind(asset.id)
            .bind(&asset.name)
            .bind(asset.asset_type.to_string())
            .bind(&asset.symbol)
            .bind(asset.quantity)
            .bind(asset.avg_buy_price)
            .bind(&asset.currency)
            .bind(asset.sort_order)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn soft_delete(&self, schema: &str, id: Uuid) -> Result<(), InvestmentError> {
        let schema = validate_tenant_schema(schema)
            .map_err(|_| InvestmentError::TenantNotProvisioned)?;
        let q = format!(
            "UPDATE {schema}.investment_assets SET is_deleted=true, deleted_at=now(), updated_at=now() WHERE id=$1 AND is_deleted=false"
        );
        sqlx::query(&q).bind(id).execute(&self.pool).await?;
        Ok(())
    }

    async fn list_unit_history(&self, schema: &str, asset_id: Uuid) -> Result<Vec<UnitHistory>, InvestmentError> {
        let schema = validate_tenant_schema(schema)
            .map_err(|_| InvestmentError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, asset_id, transaction_type, quantity_change, price_per_unit, total_value, notes, transaction_date, recorded_at \
             FROM {schema}.unit_history WHERE asset_id=$1 ORDER BY transaction_date DESC LIMIT 200"
        );
        Ok(sqlx::query_as(&q).bind(asset_id).fetch_all(&self.pool).await?)
    }

    async fn create_unit_history(&self, schema: &str, h: &UnitHistory) -> Result<(), InvestmentError> {
        let schema = validate_tenant_schema(schema)
            .map_err(|_| InvestmentError::TenantNotProvisioned)?;
        let q = format!(
            "INSERT INTO {schema}.unit_history \
             (id, asset_id, transaction_type, quantity_change, price_per_unit, total_value, notes, transaction_date, recorded_at) \
             VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now())"
        );
        sqlx::query(&q)
            .bind(h.id)
            .bind(h.asset_id)
            .bind(h.transaction_type.to_string())
            .bind(h.quantity_change)
            .bind(h.price_per_unit)
            .bind(h.total_value)
            .bind(&h.notes)
            .bind(h.transaction_date)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn list_since(&self, schema: &str, since: DateTime<Utc>) -> Result<Vec<InvestmentAsset>, InvestmentError> {
        let schema = validate_tenant_schema(schema)
            .map_err(|_| InvestmentError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, name, asset_type, symbol, quantity, avg_buy_price, currency, \
             is_deleted, deleted_at, sort_order, created_at, updated_at \
             FROM {schema}.investment_assets WHERE updated_at > $1 ORDER BY updated_at ASC"
        );
        Ok(sqlx::query_as(&q).bind(since).fetch_all(&self.pool).await?)
    }
}
