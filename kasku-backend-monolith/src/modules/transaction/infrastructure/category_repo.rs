use async_trait::async_trait;
use sqlx_postgres::PgPool;
use uuid::Uuid;

use crate::modules::transaction::domain::entity::Category;
use crate::modules::transaction::domain::error::TransactionError;
use crate::modules::transaction::domain::repository::CategoryRepository;
use crate::shared::tenant::validate_tenant_schema;

pub struct PostgresCategoryRepository {
    pool: PgPool,
}

impl PostgresCategoryRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl CategoryRepository for PostgresCategoryRepository {
    async fn list(&self, schema: &str) -> Result<Vec<Category>, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, name, icon, color, category_type, is_default, is_deleted, deleted_at, created_at, updated_at
             FROM {schema}.categories
             WHERE is_deleted = false
             ORDER BY name ASC"
        );
        let rows: Vec<Category> = sqlx::query_as(&q).fetch_all(&self.pool).await?;
        Ok(rows)
    }

    async fn get_by_id(&self, schema: &str, id: Uuid) -> Result<Option<Category>, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT id, name, icon, color, category_type, is_default, is_deleted, deleted_at, created_at, updated_at
             FROM {schema}.categories
             WHERE id = $1 AND is_deleted = false
             LIMIT 1"
        );
        let row: Option<Category> = sqlx::query_as(&q).bind(id).fetch_optional(&self.pool).await?;
        Ok(row)
    }

    async fn create(&self, schema: &str, cat: &Category) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "INSERT INTO {schema}.categories
             (id, name, icon, color, category_type, is_default, is_deleted, created_at, updated_at)
             VALUES ($1,$2,$3,$4,$5,$6,false,$7,$7)"
        );
        sqlx::query(&q)
            .bind(cat.id)
            .bind(&cat.name)
            .bind(&cat.icon)
            .bind(&cat.color)
            .bind(cat.category_type.to_string())
            .bind(cat.is_default)
            .bind(cat.created_at)
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn update(&self, schema: &str, cat: &Category) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "UPDATE {schema}.categories
             SET name = $2, icon = $3, color = $4, category_type = $5, updated_at = now()
             WHERE id = $1 AND is_deleted = false"
        );
        sqlx::query(&q)
            .bind(cat.id)
            .bind(&cat.name)
            .bind(&cat.icon)
            .bind(&cat.color)
            .bind(cat.category_type.to_string())
            .execute(&self.pool)
            .await?;
        Ok(())
    }

    async fn soft_delete(&self, schema: &str, id: Uuid) -> Result<(), TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;

        let check_q = format!(
            "SELECT is_default FROM {schema}.categories WHERE id = $1 AND is_deleted = false LIMIT 1"
        );
        let is_default: Option<bool> = sqlx::query_scalar(&check_q)
            .bind(id)
            .fetch_optional(&self.pool)
            .await?;

        match is_default {
            None => return Err(TransactionError::CategoryNotFound),
            Some(true) => return Err(TransactionError::DefaultCategoryCannotBeDeleted),
            Some(false) => {}
        }

        let q = format!(
            "UPDATE {schema}.categories SET is_deleted = true, deleted_at = now(), updated_at = now()
             WHERE id = $1"
        );
        sqlx::query(&q).bind(id).execute(&self.pool).await?;
        Ok(())
    }

    async fn has_active_transactions(&self, schema: &str, category_id: Uuid) -> Result<bool, TransactionError> {
        let schema = validate_tenant_schema(schema).map_err(|_| TransactionError::TenantNotProvisioned)?;
        let q = format!(
            "SELECT EXISTS (
                SELECT 1 FROM {schema}.transactions
                WHERE category_id = $1 AND is_deleted = false
             )"
        );
        let exists: bool = sqlx::query_scalar(&q).bind(category_id).fetch_one(&self.pool).await?;
        Ok(exists)
    }
}
