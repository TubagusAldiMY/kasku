use async_trait::async_trait;
use chrono::{DateTime, Utc};
use uuid::Uuid;

use super::entity::{Budget, BudgetWithProgress, Category, Transaction, TransactionSummary};
use super::error::TransactionError;

#[async_trait]
pub trait TransactionRepository: Send + Sync {
    async fn count_monthly(&self, schema: &str, user_id: Uuid, month: DateTime<Utc>) -> Result<i64, TransactionError>;
    async fn create(&self, schema: &str, tx: &Transaction) -> Result<(), TransactionError>;
    async fn list(&self, schema: &str, user_id: Uuid, from: DateTime<Utc>, to: DateTime<Utc>) -> Result<Vec<Transaction>, TransactionError>;
    async fn get_by_id(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<Option<Transaction>, TransactionError>;
    async fn update(&self, schema: &str, user_id: Uuid, tx: &Transaction) -> Result<(), TransactionError>;
    async fn soft_delete(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<(), TransactionError>;
    async fn get_summary(&self, schema: &str, user_id: Uuid, from: DateTime<Utc>, to: DateTime<Utc>) -> Result<TransactionSummary, TransactionError>;
    async fn list_for_export(&self, schema: &str, user_id: Uuid, from: Option<DateTime<Utc>>, to: Option<DateTime<Utc>>) -> Result<Vec<Transaction>, TransactionError>;
    async fn list_since(&self, schema: &str, since: DateTime<Utc>) -> Result<Vec<Transaction>, TransactionError>;
}

#[async_trait]
pub trait CategoryRepository: Send + Sync {
    async fn list(&self, schema: &str) -> Result<Vec<Category>, TransactionError>;
    async fn get_by_id(&self, schema: &str, id: Uuid) -> Result<Option<Category>, TransactionError>;
    async fn create(&self, schema: &str, cat: &Category) -> Result<(), TransactionError>;
    async fn update(&self, schema: &str, cat: &Category) -> Result<(), TransactionError>;
    async fn soft_delete(&self, schema: &str, id: Uuid) -> Result<(), TransactionError>;
    async fn has_active_transactions(&self, schema: &str, category_id: Uuid) -> Result<bool, TransactionError>;
}

#[async_trait]
pub trait BudgetRepository: Send + Sync {
    async fn count(&self, schema: &str, user_id: Uuid) -> Result<i64, TransactionError>;
    async fn create(&self, schema: &str, b: &Budget) -> Result<(), TransactionError>;
    async fn list(&self, schema: &str, user_id: Uuid) -> Result<Vec<BudgetWithProgress>, TransactionError>;
    async fn get_by_id(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<Option<BudgetWithProgress>, TransactionError>;
    async fn update(&self, schema: &str, b: &Budget) -> Result<(), TransactionError>;
    async fn soft_delete(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<(), TransactionError>;
}
