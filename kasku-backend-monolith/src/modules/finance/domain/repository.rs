use async_trait::async_trait;
use chrono::DateTime;
use chrono::Utc;
use uuid::Uuid;

use super::entity::{BalanceHistory, FinancialAccount};
use super::error::FinanceError;

#[async_trait]
pub trait FinancialAccountRepository: Send + Sync {
    async fn list(&self, tenant_schema: &str, user_id: Uuid) -> Result<Vec<FinancialAccount>, FinanceError>;
    async fn find_by_id(&self, tenant_schema: &str, id: Uuid, user_id: Uuid) -> Result<Option<FinancialAccount>, FinanceError>;
    async fn create(&self, tenant_schema: &str, account: &FinancialAccount) -> Result<(), FinanceError>;
    async fn update(&self, tenant_schema: &str, account: &FinancialAccount) -> Result<(), FinanceError>;
    async fn soft_delete(&self, tenant_schema: &str, id: Uuid, user_id: Uuid) -> Result<(), FinanceError>;
    async fn list_history(&self, tenant_schema: &str, account_id: Uuid) -> Result<Vec<BalanceHistory>, FinanceError>;
    async fn count(&self, tenant_schema: &str, user_id: Uuid) -> Result<i64, FinanceError>;
    async fn list_since(&self, tenant_schema: &str, since: DateTime<Utc>) -> Result<Vec<FinancialAccount>, FinanceError>;
}
