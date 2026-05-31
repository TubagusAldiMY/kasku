use async_trait::async_trait;
use sqlx::Row;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::transaction::domain::entity::{Budget, BudgetPeriodType, BudgetWithProgress};
use crate::modules::transaction::domain::error::TransactionError;
use crate::modules::transaction::domain::repository::BudgetRepository;
use crate::shared::tenant::validate_tenant_schema;

pub struct PostgresBudgetRepository {
    pool: PgPool,
}

impl PostgresBudgetRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

fn row_to_budget_with_progress(row: &sqlx::postgres::PgRow) -> Result<BudgetWithProgress, sqlx::Error> {
    let period_str: String = row.try_get("period_type")?;
    let period_type = period_str.parse::<BudgetPeriodType>().map_err(|e| {
        sqlx::Error::ColumnDecode {
            index: "period_type".to_string(),
            source: Box::new(std::io::Error::new(std::io::ErrorKind::InvalidData, e)),
        }
    })?;
    let budget = Budget {
        id: row.try_get("id")?,
        user_id: row.try_get("user_id")?,
        sync_id: row.try_get("sync_id")?,
        name: row.try_get("name")?,
        limit_idr: row.try_get("limit_idr")?,
        category_id: row.try_get("category_id")?,
        period_type,
        start_date: row.try_get("start_date")?,
        end_date: row.try_get("end_date")?,
        alert_threshold: row.try_get("alert_threshold")?,
        daily_limit_enabled: row.try_get("daily_limit_enabled")?,
        is_deleted: row.try_get("is_deleted")?,
        deleted_at: row.try_get("deleted_at")?,
        created_at: row.try_get("created_at")?,
        updated_at: row.try_get("updated_at")?,
    };
    Ok(BudgetWithProgress {
        budget,
        spent_idr: row.try_get("spent_idr")?,
        category_name: row.try_get("category_name")?,
    })
}

#[async_trait]
impl BudgetRepository for PostgresBudgetRepository {
    async fn count(&self, schema: &str, user_id: Uuid) -> Result<i64, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT COUNT(*) FROM {schema}.budgets WHERE user_id = $1 AND is_deleted = false"
        );
        let count: i64 = sqlx::query_scalar(&q).bind(user_id).fetch_one(&self.pool).await?;
        Ok(count)
    }

    async fn create(&self, schema: &str, b: &Budget) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "INSERT INTO {schema}.budgets
             (id, user_id, sync_id, name, limit_idr, category_id, period_type, start_date, end_date,
              alert_threshold, daily_limit_enabled, is_deleted, created_at, updated_at)
             VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,false,$12,$12)"
        );
        sqlx::query(&q)
            .bind(b.id)
            .bind(b.user_id)
            .bind(&b.sync_id)
            .bind(&b.name)
            .bind(b.limit_idr)
            .bind(b.category_id)
            .bind(b.period_type.to_string())
            .bind(b.start_date)
            .bind(b.end_date)
            .bind(b.alert_threshold)
            .bind(b.daily_limit_enabled)
            .bind(b.created_at)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn list(&self, schema: &str, user_id: Uuid) -> Result<Vec<BudgetWithProgress>, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT b.id, b.user_id, b.sync_id, b.name, b.limit_idr, b.category_id, b.period_type,
                    b.start_date, b.end_date, b.alert_threshold, b.daily_limit_enabled,
                    b.is_deleted, b.deleted_at, b.created_at, b.updated_at,
                    COALESCE((
                        SELECT SUM(t.amount_idr)
                        FROM {schema}.transactions t
                        WHERE t.budget_id = b.id AND t.is_deleted = false
                          AND t.transaction_date >= b.start_date
                          AND (b.end_date IS NULL OR t.transaction_date <= b.end_date)
                          AND t.transaction_type = 'EXPENSE'
                    ), 0) AS spent_idr,
                    COALESCE(c.name, '') AS category_name
             FROM {schema}.budgets b
             LEFT JOIN {schema}.categories c ON c.id = b.category_id
             WHERE b.user_id = $1 AND b.is_deleted = false
             ORDER BY b.created_at DESC"
        );
        let rows = sqlx::query(&q)
            .bind(user_id)
            .fetch_all(&self.pool)
            .await?;

        rows.iter()
            .map(|r| row_to_budget_with_progress(r).map_err(TransactionError::from))
            .collect()
    }

    async fn get_by_id(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<Option<BudgetWithProgress>, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT b.id, b.user_id, b.sync_id, b.name, b.limit_idr, b.category_id, b.period_type,
                    b.start_date, b.end_date, b.alert_threshold, b.daily_limit_enabled,
                    b.is_deleted, b.deleted_at, b.created_at, b.updated_at,
                    COALESCE((
                        SELECT SUM(t.amount_idr)
                        FROM {schema}.transactions t
                        WHERE t.budget_id = b.id AND t.is_deleted = false
                          AND t.transaction_date >= b.start_date
                          AND (b.end_date IS NULL OR t.transaction_date <= b.end_date)
                          AND t.transaction_type = 'EXPENSE'
                    ), 0) AS spent_idr,
                    COALESCE(c.name, '') AS category_name
             FROM {schema}.budgets b
             LEFT JOIN {schema}.categories c ON c.id = b.category_id
             WHERE b.id = $1 AND b.user_id = $2 AND b.is_deleted = false
             LIMIT 1"
        );
        let row = sqlx::query(&q)
            .bind(id)
            .bind(user_id)
            .fetch_optional(&self.pool)
            .await?;

        match row {
            Some(r) => Ok(Some(row_to_budget_with_progress(&r).map_err(TransactionError::from)?)),
            None => Ok(None),
        }
    }

    async fn update(&self, schema: &str, b: &Budget) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "UPDATE {schema}.budgets
             SET name = $2, limit_idr = $3, category_id = $4, period_type = $5,
                 start_date = $6, end_date = $7, alert_threshold = $8, daily_limit_enabled = $9,
                 updated_at = now()
             WHERE id = $1 AND is_deleted = false"
        );
        sqlx::query(&q)
            .bind(b.id)
            .bind(&b.name)
            .bind(b.limit_idr)
            .bind(b.category_id)
            .bind(b.period_type.to_string())
            .bind(b.start_date)
            .bind(b.end_date)
            .bind(b.alert_threshold)
            .bind(b.daily_limit_enabled)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn soft_delete(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "UPDATE {schema}.budgets SET is_deleted = true, deleted_at = now(), updated_at = now()
             WHERE id = $1 AND user_id = $2 AND is_deleted = false"
        );
        sqlx::query(&q).bind(id).bind(user_id).execute(&self.pool).await?;
        Ok(())
    }
}
