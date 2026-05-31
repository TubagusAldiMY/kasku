use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::finance::domain::{entity::{BalanceHistory, FinancialAccount}, error::FinanceError, repository::FinancialAccountRepository};
use crate::shared::tenant::validate_tenant_schema;

pub struct PostgresFinancialAccountRepository {
    pool: PgPool,
}

impl PostgresFinancialAccountRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl FinancialAccountRepository for PostgresFinancialAccountRepository {
    async fn list(&self, tenant_schema: &str, user_id: Uuid) -> Result<Vec<FinancialAccount>, FinanceError> {
        let schema = validate_tenant_schema(tenant_schema)
            .map_err(|_| FinanceError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, user_id, name, account_type, balance, initial_balance, currency, color, icon,
             is_default, is_deleted, deleted_at, created_at, updated_at
             FROM {schema}.financial_accounts WHERE user_id = $1 AND is_deleted = false ORDER BY created_at ASC"
        );
        let accounts: Vec<FinancialAccount> = sqlx::query_as(&q)
            .bind(user_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(accounts)
    }

    async fn find_by_id(&self, tenant_schema: &str, id: Uuid, user_id: Uuid) -> Result<Option<FinancialAccount>, FinanceError> {
        let schema = validate_tenant_schema(tenant_schema)
            .map_err(|_| FinanceError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, user_id, name, account_type, balance, initial_balance, currency, color, icon,
             is_default, is_deleted, deleted_at, created_at, updated_at
             FROM {schema}.financial_accounts WHERE id = $1 AND user_id = $2 AND is_deleted = false LIMIT 1"
        );
        let account: Option<FinancialAccount> = sqlx::query_as(&q)
            .bind(id)
            .bind(user_id)
            .fetch_optional(&self.pool)
            .await?;
        Ok(account)
    }

    async fn create(&self, tenant_schema: &str, account: &FinancialAccount) -> Result<(), FinanceError> {
        let schema = validate_tenant_schema(tenant_schema)
            .map_err(|_| FinanceError::TenantNotProvisioned)?;
        let q = format!(
            "INSERT INTO {schema}.financial_accounts
             (id, user_id, name, account_type, balance, initial_balance, currency, color, icon, is_default, is_deleted, created_at, updated_at)
             VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,false,$11,$11)"
        );
        sqlx::query(&q)
            .bind(account.id)
            .bind(account.user_id)
            .bind(&account.name)
            .bind(&account.account_type)
            .bind(account.balance)
            .bind(account.initial_balance)
            .bind(&account.currency)
            .bind(&account.color)
            .bind(&account.icon)
            .bind(account.is_default)
            .bind(account.created_at)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn update(&self, tenant_schema: &str, account: &FinancialAccount) -> Result<(), FinanceError> {
        let schema = validate_tenant_schema(tenant_schema)
            .map_err(|_| FinanceError::TenantNotProvisioned)?;
        let now = Utc::now();
        let q = format!(
            "UPDATE {schema}.financial_accounts
             SET name=$2, account_type=$3, color=$4, icon=$5, is_default=$6, updated_at=$7
             WHERE id=$1 AND user_id=$8 AND is_deleted=false"
        );
        sqlx::query(&q)
            .bind(account.id)
            .bind(&account.name)
            .bind(&account.account_type)
            .bind(&account.color)
            .bind(&account.icon)
            .bind(account.is_default)
            .bind(now)
            .bind(account.user_id)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn soft_delete(&self, tenant_schema: &str, id: Uuid, user_id: Uuid) -> Result<(), FinanceError> {
        let schema = validate_tenant_schema(tenant_schema)
            .map_err(|_| FinanceError::TenantNotProvisioned)?;
        let now = Utc::now();
        let q = format!(
            "UPDATE {schema}.financial_accounts SET is_deleted=true, deleted_at=$3, updated_at=$3 WHERE id=$1 AND user_id=$2"
        );
        sqlx::query(&q).bind(id).bind(user_id).bind(now).execute(&self.pool).await?;
        Ok(())
    }

    async fn list_history(&self, tenant_schema: &str, account_id: Uuid) -> Result<Vec<BalanceHistory>, FinanceError> {
        let schema = validate_tenant_schema(tenant_schema)
            .map_err(|_| FinanceError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, account_id, amount, balance, note, created_at FROM {schema}.balance_history WHERE account_id=$1 ORDER BY created_at DESC LIMIT 100"
        );
        let rows: Vec<BalanceHistory> = sqlx::query_as(&q).bind(account_id).fetch_all(&self.pool).await?;
        Ok(rows)
    }

    async fn count(&self, tenant_schema: &str, user_id: Uuid) -> Result<i64, FinanceError> {
        let schema = validate_tenant_schema(tenant_schema)
            .map_err(|_| FinanceError::TenantNotProvisioned)?;
        let q = format!("SELECT COUNT(*) FROM {schema}.financial_accounts WHERE user_id=$1 AND is_deleted=false");
        let count: i64 = sqlx::query_scalar(&q).bind(user_id).fetch_one(&self.pool).await?;
        Ok(count)
    }

    async fn list_since(&self, tenant_schema: &str, since: DateTime<Utc>) -> Result<Vec<FinancialAccount>, FinanceError> {
        let schema = validate_tenant_schema(tenant_schema)
            .map_err(|_| FinanceError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, user_id, name, account_type, balance, initial_balance, currency, color, icon,
             is_default, is_deleted, deleted_at, created_at, updated_at
             FROM {schema}.financial_accounts WHERE updated_at > $1 ORDER BY updated_at ASC"
        );
        let accounts: Vec<FinancialAccount> = sqlx::query_as(&q).bind(since).fetch_all(&self.pool).await?;
        Ok(accounts)
    }
}
