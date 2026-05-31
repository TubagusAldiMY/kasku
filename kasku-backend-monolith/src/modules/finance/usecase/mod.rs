use std::sync::Arc;
use uuid::Uuid;
use crate::modules::finance::domain::{entity::{BalanceHistory, FinancialAccount}, error::FinanceError, repository::FinancialAccountRepository};
use crate::shared::middleware::tier_inject::TierLimits;
use chrono::Utc;

pub struct FinanceUseCases {
    pub account_repo: Arc<dyn FinancialAccountRepository>,
}

impl FinanceUseCases {
    pub fn new(account_repo: Arc<dyn FinancialAccountRepository>) -> Self {
        Self { account_repo }
    }

    pub async fn list_accounts(&self, tenant_schema: &str, user_id: Uuid) -> Result<Vec<FinancialAccount>, FinanceError> {
        self.account_repo.list(tenant_schema, user_id).await
    }

    pub async fn create_account(
        &self, tenant_schema: &str, user_id: Uuid, name: String,
        account_type: String, currency: String, color: String, icon: String,
        initial_balance: i64, is_default: bool, limits: &TierLimits,
    ) -> Result<FinancialAccount, FinanceError> {
        if limits.max_financial_accounts != -1 {
            let count = self.account_repo.count(tenant_schema, user_id).await?;
            if count >= limits.max_financial_accounts {
                return Err(FinanceError::AccountLimitReached);
            }
        }
        let now = Utc::now();
        let account = FinancialAccount {
            id: Uuid::new_v4(),
            user_id,
            name,
            account_type,
            balance: initial_balance,
            initial_balance,
            currency,
            color,
            icon,
            is_default,
            is_deleted: false,
            deleted_at: None,
            created_at: now,
            updated_at: now,
        };
        self.account_repo.create(tenant_schema, &account).await?;
        Ok(account)
    }

    pub async fn update_account(&self, tenant_schema: &str, id: Uuid, user_id: Uuid, name: Option<String>, color: Option<String>, icon: Option<String>, is_default: Option<bool>) -> Result<FinancialAccount, FinanceError> {
        let mut account = self.account_repo.find_by_id(tenant_schema, id, user_id).await?
            .ok_or(FinanceError::AccountNotFound)?;
        if let Some(n) = name { account.name = n; }
        if let Some(c) = color { account.color = c; }
        if let Some(i) = icon { account.icon = i; }
        if let Some(d) = is_default { account.is_default = d; }
        self.account_repo.update(tenant_schema, &account).await?;
        Ok(account)
    }

    pub async fn delete_account(&self, tenant_schema: &str, id: Uuid, user_id: Uuid) -> Result<(), FinanceError> {
        let _ = self.account_repo.find_by_id(tenant_schema, id, user_id).await?
            .ok_or(FinanceError::AccountNotFound)?;
        self.account_repo.soft_delete(tenant_schema, id, user_id).await
    }

    pub async fn get_balance_history(&self, tenant_schema: &str, account_id: Uuid, user_id: Uuid) -> Result<Vec<BalanceHistory>, FinanceError> {
        let _ = self.account_repo.find_by_id(tenant_schema, account_id, user_id).await?
            .ok_or(FinanceError::AccountNotFound)?;
        self.account_repo.list_history(tenant_schema, account_id).await
    }
}
