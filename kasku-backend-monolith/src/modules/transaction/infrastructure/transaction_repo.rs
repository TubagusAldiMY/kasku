use async_trait::async_trait;
use chrono::{DateTime, Datelike, Utc};
use sqlx::Acquire;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::transaction::domain::entity::{Transaction, TransactionSummary};
use crate::modules::transaction::domain::error::TransactionError;
use crate::modules::transaction::domain::repository::TransactionRepository;
use crate::shared::tenant::validate_tenant_schema;

pub struct PostgresTransactionRepository {
    pool: PgPool,
}

impl PostgresTransactionRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    async fn recalculate_balance(
        &self,
        tx: &mut sqlx::Transaction<'_, sqlx::Postgres>,
        schema: &str,
        account_id: Uuid,
    ) -> Result<(), TransactionError> {
        let q = format!(
            "UPDATE {schema}.financial_accounts fa SET
                balance = fa.initial_balance
                    + COALESCE((
                        SELECT SUM(CASE
                            WHEN t.transaction_type = 'INCOME' THEN t.amount_idr
                            WHEN t.transaction_type = 'EXPENSE' THEN -t.amount_idr
                            WHEN t.transaction_type = 'TRANSFER' THEN -t.amount_idr
                            ELSE 0 END)
                        FROM {schema}.transactions t
                        WHERE t.account_id = $1 AND t.is_deleted = false
                    ), 0)
                    + COALESCE((
                        SELECT SUM(t.amount_idr)
                        FROM {schema}.transactions t
                        WHERE t.to_account_id = $1 AND t.is_deleted = false AND t.transaction_type = 'TRANSFER'
                    ), 0),
                updated_at = now()
            WHERE fa.id = $1 AND fa.is_deleted = false"
        );
        sqlx::query(&q)
            .bind(account_id)
            .execute(&mut **tx)
            .await
            .map_err(TransactionError::from)?;
        Ok(())
    }
}

#[async_trait]
impl TransactionRepository for PostgresTransactionRepository {
    async fn count_monthly(&self, schema: &str, user_id: Uuid, month: DateTime<Utc>) -> Result<i64, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let month_start = month;
        let month_end = {
            let next = month.date_naive().succ_opt()
                .map(|_| {
                    let year = month.date_naive().year();
                    let m = month.date_naive().month();
                    if m == 12 {
                        chrono::NaiveDate::from_ymd_opt(year + 1, 1, 1).unwrap()
                    } else {
                        chrono::NaiveDate::from_ymd_opt(year, m + 1, 1).unwrap()
                    }
                })
                .unwrap_or_else(|| month.date_naive());
            next.and_hms_opt(0, 0, 0).unwrap().and_utc()
        };
        let q = format!(
            "SELECT COUNT(*) FROM {schema}.transactions t
             JOIN {schema}.financial_accounts a ON t.account_id = a.id
             WHERE a.user_id = $1 AND t.is_deleted = false
               AND t.transaction_date >= $2 AND t.transaction_date < $3"
        );
        let count: i64 = sqlx::query_scalar(&q)
            .bind(user_id)
            .bind(month_start)
            .bind(month_end)
            .fetch_one(&self.pool)
            .await?;
        Ok(count)
    }

    async fn create(&self, schema: &str, transaction: &Transaction) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let mut db_tx = self.pool.begin().await?;

        let q = format!(
            "INSERT INTO {schema}.transactions
             (id, sync_id, account_id, category_id, budget_id, transaction_type, amount_idr,
              transaction_date, notes, to_account_id, is_deleted, created_at, updated_at)
             VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,false,$11,$11)
             ON CONFLICT (sync_id) DO NOTHING"
        );
        sqlx::query(&q)
            .bind(transaction.id)
            .bind(&transaction.sync_id)
            .bind(transaction.account_id)
            .bind(transaction.category_id)
            .bind(transaction.budget_id)
            .bind(transaction.transaction_type.to_string())
            .bind(transaction.amount_idr)
            .bind(transaction.transaction_date)
            .bind(&transaction.notes)
            .bind(transaction.to_account_id)
            .bind(transaction.created_at)
            .execute(&mut *db_tx)
            .await?;

        self.recalculate_balance(&mut db_tx, schema, transaction.account_id).await?;
        if let Some(to_id) = transaction.to_account_id {
            self.recalculate_balance(&mut db_tx, schema, to_id).await?;
        }

        db_tx.commit().await?;
        Ok(())
    }

    async fn list(&self, schema: &str, user_id: Uuid, from: DateTime<Utc>, to: DateTime<Utc>) -> Result<Vec<Transaction>, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT t.id, t.sync_id, t.account_id, t.category_id, t.budget_id,
                    t.transaction_type, t.amount_idr, t.transaction_date, t.notes,
                    t.to_account_id, t.is_deleted, t.deleted_at, t.created_at, t.updated_at
             FROM {schema}.transactions t
             JOIN {schema}.financial_accounts a ON t.account_id = a.id
             WHERE a.user_id = $1 AND t.is_deleted = false
               AND t.transaction_date >= $2 AND t.transaction_date <= $3
             ORDER BY t.transaction_date DESC"
        );
        let rows: Vec<Transaction> = sqlx::query_as(&q)
            .bind(user_id)
            .bind(from)
            .bind(to)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows)
    }

    async fn get_by_id(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<Option<Transaction>, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT t.id, t.sync_id, t.account_id, t.category_id, t.budget_id,
                    t.transaction_type, t.amount_idr, t.transaction_date, t.notes,
                    t.to_account_id, t.is_deleted, t.deleted_at, t.created_at, t.updated_at
             FROM {schema}.transactions t
             JOIN {schema}.financial_accounts a ON t.account_id = a.id
             WHERE t.id = $1 AND a.user_id = $2 AND t.is_deleted = false
             LIMIT 1"
        );
        let row: Option<Transaction> = sqlx::query_as(&q)
            .bind(id)
            .bind(user_id)
            .fetch_optional(&self.pool)
            .await?;
        Ok(row)
    }

    async fn update(&self, schema: &str, user_id: Uuid, transaction: &Transaction) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let mut db_tx = self.pool.begin().await?;

        let q = format!(
            "UPDATE {schema}.transactions t SET
                account_id = $3,
                category_id = $4,
                budget_id = $5,
                transaction_type = $6,
                amount_idr = $7,
                transaction_date = $8,
                notes = $9,
                to_account_id = $10,
                updated_at = now()
             FROM {schema}.financial_accounts a
             WHERE t.id = $1 AND t.account_id = a.id AND a.user_id = $2 AND t.is_deleted = false"
        );
        sqlx::query(&q)
            .bind(transaction.id)
            .bind(user_id)
            .bind(transaction.account_id)
            .bind(transaction.category_id)
            .bind(transaction.budget_id)
            .bind(transaction.transaction_type.to_string())
            .bind(transaction.amount_idr)
            .bind(transaction.transaction_date)
            .bind(&transaction.notes)
            .bind(transaction.to_account_id)
            .execute(&mut *db_tx)
            .await?;

        self.recalculate_balance(&mut db_tx, schema, transaction.account_id).await?;
        if let Some(to_id) = transaction.to_account_id {
            self.recalculate_balance(&mut db_tx, schema, to_id).await?;
        }

        db_tx.commit().await?;
        Ok(())
    }

    async fn soft_delete(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let mut db_tx = self.pool.begin().await?;

        // Fetch account_id and to_account_id before deleting so we can recalculate
        let fetch_q = format!(
            "SELECT t.account_id, t.to_account_id FROM {schema}.transactions t
             JOIN {schema}.financial_accounts a ON t.account_id = a.id
             WHERE t.id = $1 AND a.user_id = $2 AND t.is_deleted = false
             LIMIT 1"
        );
        let row: Option<(Uuid, Option<Uuid>)> = sqlx::query_as(&fetch_q)
            .bind(id)
            .bind(user_id)
            .fetch_optional(&mut *db_tx)
            .await?;

        let (account_id, to_account_id) = match row {
            Some(r) => r,
            None => return Err(TransactionError::NotFound),
        };

        let q = format!(
            "UPDATE {schema}.transactions SET is_deleted = true, deleted_at = now(), updated_at = now()
             WHERE id = $1"
        );
        sqlx::query(&q).bind(id).execute(&mut *db_tx).await?;

        self.recalculate_balance(&mut db_tx, schema, account_id).await?;
        if let Some(to_id) = to_account_id {
            self.recalculate_balance(&mut db_tx, schema, to_id).await?;
        }

        db_tx.commit().await?;
        Ok(())
    }

    async fn get_summary(&self, schema: &str, user_id: Uuid, from: DateTime<Utc>, to: DateTime<Utc>) -> Result<TransactionSummary, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT
                COALESCE(SUM(CASE WHEN t.transaction_type = 'INCOME' THEN t.amount_idr ELSE 0 END), 0) AS total_income,
                COALESCE(SUM(CASE WHEN t.transaction_type = 'EXPENSE' THEN t.amount_idr ELSE 0 END), 0) AS total_expense
             FROM {schema}.transactions t
             JOIN {schema}.financial_accounts a ON t.account_id = a.id
             WHERE a.user_id = $1 AND t.is_deleted = false
               AND t.transaction_date >= $2 AND t.transaction_date <= $3"
        );
        let summary: TransactionSummary = sqlx::query_as(&q)
            .bind(user_id)
            .bind(from)
            .bind(to)
            .fetch_one(&self.pool)
            .await?;
        Ok(summary)
    }

    async fn list_for_export(&self, schema: &str, user_id: Uuid, from: Option<DateTime<Utc>>, to: Option<DateTime<Utc>>) -> Result<Vec<Transaction>, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let mut conditions = vec![
            "a.user_id = $1".to_string(),
            "t.is_deleted = false".to_string(),
        ];
        let mut bindings: Vec<Box<dyn Fn(sqlx::query::Query<sqlx::Postgres, sqlx::postgres::PgArguments>) -> sqlx::query::Query<sqlx::Postgres, sqlx::postgres::PgArguments> + Send>> = vec![];
        let _ = bindings; // unused - we use a simpler approach below

        // Use optional date filtering inline
        let date_clause = match (from, to) {
            (Some(f), Some(t)) => format!("AND t.transaction_date >= '{}' AND t.transaction_date <= '{}'", f.to_rfc3339(), t.to_rfc3339()),
            (Some(f), None) => format!("AND t.transaction_date >= '{}'", f.to_rfc3339()),
            (None, Some(t)) => format!("AND t.transaction_date <= '{}'", t.to_rfc3339()),
            (None, None) => String::new(),
        };

        let q = format!(
            "SELECT t.id, t.sync_id, t.account_id, t.category_id, t.budget_id,
                    t.transaction_type, t.amount_idr, t.transaction_date, t.notes,
                    t.to_account_id, t.is_deleted, t.deleted_at, t.created_at, t.updated_at
             FROM {schema}.transactions t
             JOIN {schema}.financial_accounts a ON t.account_id = a.id
             WHERE a.user_id = $1 AND t.is_deleted = false {date_clause}
             ORDER BY t.transaction_date DESC"
        );
        let rows: Vec<Transaction> = sqlx::query_as(&q)
            .bind(user_id)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows)
    }

    async fn list_since(&self, schema: &str, since: DateTime<Utc>) -> Result<Vec<Transaction>, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, sync_id, account_id, category_id, budget_id,
                    transaction_type, amount_idr, transaction_date, notes,
                    to_account_id, is_deleted, deleted_at, created_at, updated_at
             FROM {schema}.transactions
             WHERE updated_at > $1
             ORDER BY updated_at ASC"
        );
        let rows: Vec<Transaction> = sqlx::query_as(&q)
            .bind(since)
            .fetch_all(&self.pool)
            .await?;
        Ok(rows)
    }
}
