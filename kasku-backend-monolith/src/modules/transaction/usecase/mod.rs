use std::sync::Arc;
use chrono::{DateTime, Datelike, Utc};
use uuid::Uuid;

use crate::app_error::AppError;
use crate::shared::middleware::tier_inject::TierLimits;
use crate::modules::transaction::domain::entity::{
    Budget, BudgetPeriodType, BudgetWithProgress, Category, CategoryType, Transaction,
    TransactionSummary, TransactionType,
};
use crate::modules::transaction::domain::error::TransactionError;
use crate::modules::transaction::domain::repository::{
    BudgetRepository, CategoryRepository, TransactionRepository,
};

pub struct TransactionUseCases {
    pub tx_repo: Arc<dyn TransactionRepository>,
    pub cat_repo: Arc<dyn CategoryRepository>,
    pub budget_repo: Arc<dyn BudgetRepository>,
}

impl TransactionUseCases {
    pub fn new(
        tx_repo: Arc<dyn TransactionRepository>,
        cat_repo: Arc<dyn CategoryRepository>,
        budget_repo: Arc<dyn BudgetRepository>,
    ) -> Self {
        Self { tx_repo, cat_repo, budget_repo }
    }
}

fn tx_err_to_app(e: TransactionError) -> AppError {
    match e {
        TransactionError::NotFound => AppError::NotFound,
        TransactionError::CategoryNotFound => AppError::NotFound,
        TransactionError::BudgetNotFound => AppError::NotFound,
        TransactionError::AccountNotFound => AppError::NotFound,
        TransactionError::DefaultCategoryCannotBeDeleted => {
            AppError::Unprocessable(e.to_string())
        }
        TransactionError::CategoryHasTransactions => {
            AppError::Conflict(e.to_string())
        }
        TransactionError::TransactionLimitReached | TransactionError::BudgetLimitReached => {
            AppError::TierLimitExceeded(e.to_string())
        }
        TransactionError::ExportNotAllowed => {
            AppError::TierLimitExceeded(e.to_string())
        }
        TransactionError::InvalidInput(msg) => AppError::Validation(msg),
        TransactionError::InsufficientBalance => AppError::Unprocessable(e.to_string()),
        TransactionError::TenantNotProvisioned => AppError::TenantNotProvisioned,
        TransactionError::Internal(msg) => AppError::Internal(anyhow::anyhow!(msg)),
    }
}

// ---------- Transactions ----------

impl TransactionUseCases {
    pub async fn list_transactions(
        &self,
        schema: &str,
        user_id: Uuid,
        from: DateTime<Utc>,
        to: DateTime<Utc>,
    ) -> Result<Vec<Transaction>, AppError> {
        self.tx_repo.list(schema, user_id, from, to).await.map_err(tx_err_to_app)
    }

    pub async fn create_transaction(
        &self,
        schema: &str,
        user_id: Uuid,
        account_id: Uuid,
        category_id: Option<Uuid>,
        budget_id: Option<Uuid>,
        transaction_type: TransactionType,
        amount_idr: i64,
        transaction_date: DateTime<Utc>,
        notes: String,
        to_account_id: Option<Uuid>,
        sync_id: Option<String>,
        limits: &TierLimits,
    ) -> Result<Transaction, AppError> {
        if limits.max_transactions_per_month != -1 {
            let month_start = transaction_date
                .date_naive()
                .with_day(1)
                .unwrap()
                .and_hms_opt(0, 0, 0)
                .unwrap()
                .and_utc();
            let count = self.tx_repo.count_monthly(schema, user_id, month_start).await.map_err(tx_err_to_app)?;
            if count >= limits.max_transactions_per_month {
                return Err(AppError::TierLimitExceeded(
                    TransactionError::TransactionLimitReached.to_string(),
                ));
            }
        }

        let now = Utc::now();
        let id = Uuid::new_v4();
        let final_sync_id = sync_id.unwrap_or_else(|| id.to_string());

        let tx = Transaction {
            id,
            sync_id: final_sync_id,
            account_id,
            category_id,
            budget_id,
            transaction_type,
            amount_idr,
            transaction_date,
            notes,
            to_account_id,
            is_deleted: false,
            deleted_at: None,
            created_at: now,
            updated_at: now,
        };

        self.tx_repo.create(schema, &tx).await.map_err(tx_err_to_app)?;
        Ok(tx)
    }

    pub async fn get_transaction(
        &self,
        schema: &str,
        id: Uuid,
        user_id: Uuid,
    ) -> Result<Transaction, AppError> {
        self.tx_repo
            .get_by_id(schema, id, user_id)
            .await
            .map_err(tx_err_to_app)?
            .ok_or(AppError::NotFound)
    }

    pub async fn update_transaction(
        &self,
        schema: &str,
        user_id: Uuid,
        id: Uuid,
        account_id: Option<Uuid>,
        category_id: Option<Uuid>,
        budget_id: Option<Uuid>,
        transaction_type: Option<TransactionType>,
        amount_idr: Option<i64>,
        transaction_date: Option<DateTime<Utc>>,
        notes: Option<String>,
        to_account_id: Option<Uuid>,
    ) -> Result<Transaction, AppError> {
        let mut tx = self
            .tx_repo
            .get_by_id(schema, id, user_id)
            .await
            .map_err(tx_err_to_app)?
            .ok_or(AppError::NotFound)?;

        if let Some(v) = account_id { tx.account_id = v; }
        if let Some(v) = category_id { tx.category_id = Some(v); }
        if let Some(v) = budget_id { tx.budget_id = Some(v); }
        if let Some(v) = transaction_type { tx.transaction_type = v; }
        if let Some(v) = amount_idr { tx.amount_idr = v; }
        if let Some(v) = transaction_date { tx.transaction_date = v; }
        if let Some(v) = notes { tx.notes = v; }
        if let Some(v) = to_account_id { tx.to_account_id = Some(v); }

        self.tx_repo.update(schema, user_id, &tx).await.map_err(tx_err_to_app)?;
        Ok(tx)
    }

    pub async fn delete_transaction(
        &self,
        schema: &str,
        id: Uuid,
        user_id: Uuid,
    ) -> Result<(), AppError> {
        self.tx_repo.soft_delete(schema, id, user_id).await.map_err(tx_err_to_app)
    }

    pub async fn get_summary(
        &self,
        schema: &str,
        user_id: Uuid,
        from: DateTime<Utc>,
        to: DateTime<Utc>,
    ) -> Result<TransactionSummary, AppError> {
        self.tx_repo.get_summary(schema, user_id, from, to).await.map_err(tx_err_to_app)
    }

    pub async fn export_csv(
        &self,
        schema: &str,
        user_id: Uuid,
        from: Option<DateTime<Utc>>,
        to: Option<DateTime<Utc>>,
        limits: &TierLimits,
    ) -> Result<String, AppError> {
        if !limits.export_csv_enabled {
            return Err(AppError::TierLimitExceeded(
                TransactionError::ExportNotAllowed.to_string(),
            ));
        }
        let rows = self
            .tx_repo
            .list_for_export(schema, user_id, from, to)
            .await
            .map_err(tx_err_to_app)?;

        let mut csv = String::from("id,sync_id,account_id,category_id,budget_id,transaction_type,amount_idr,transaction_date,notes,to_account_id\n");
        for r in rows {
            csv.push_str(&format!(
                "{},{},{},{},{},{},{},{},{},{}\n",
                r.id,
                r.sync_id,
                r.account_id,
                r.category_id.map(|u| u.to_string()).unwrap_or_default(),
                r.budget_id.map(|u| u.to_string()).unwrap_or_default(),
                r.transaction_type,
                r.amount_idr,
                r.transaction_date.to_rfc3339(),
                r.notes.replace(',', ";"),
                r.to_account_id.map(|u| u.to_string()).unwrap_or_default(),
            ));
        }
        Ok(csv)
    }
}

// ---------- Categories ----------

impl TransactionUseCases {
    pub async fn list_categories(&self, schema: &str) -> Result<Vec<Category>, AppError> {
        self.cat_repo.list(schema).await.map_err(tx_err_to_app)
    }

    pub async fn create_category(
        &self,
        schema: &str,
        name: String,
        icon: String,
        color: String,
        category_type: CategoryType,
    ) -> Result<Category, AppError> {
        let now = Utc::now();
        let cat = Category {
            id: Uuid::new_v4(),
            name,
            icon,
            color,
            category_type,
            is_default: false,
            is_deleted: false,
            deleted_at: None,
            created_at: now,
            updated_at: now,
        };
        self.cat_repo.create(schema, &cat).await.map_err(tx_err_to_app)?;
        Ok(cat)
    }

    pub async fn update_category(
        &self,
        schema: &str,
        id: Uuid,
        name: Option<String>,
        icon: Option<String>,
        color: Option<String>,
        category_type: Option<CategoryType>,
    ) -> Result<Category, AppError> {
        let mut cat = self
            .cat_repo
            .get_by_id(schema, id)
            .await
            .map_err(tx_err_to_app)?
            .ok_or(AppError::NotFound)?;

        if let Some(v) = name { cat.name = v; }
        if let Some(v) = icon { cat.icon = v; }
        if let Some(v) = color { cat.color = v; }
        if let Some(v) = category_type { cat.category_type = v; }

        self.cat_repo.update(schema, &cat).await.map_err(tx_err_to_app)?;
        Ok(cat)
    }

    pub async fn delete_category(&self, schema: &str, id: Uuid) -> Result<(), AppError> {
        let has_tx = self
            .cat_repo
            .has_active_transactions(schema, id)
            .await
            .map_err(tx_err_to_app)?;
        if has_tx {
            return Err(AppError::Conflict(
                TransactionError::CategoryHasTransactions.to_string(),
            ));
        }
        self.cat_repo.soft_delete(schema, id).await.map_err(tx_err_to_app)
    }
}

// ---------- Budgets ----------

impl TransactionUseCases {
    pub async fn list_budgets(&self, schema: &str, user_id: Uuid) -> Result<Vec<BudgetWithProgress>, AppError> {
        self.budget_repo.list(schema, user_id).await.map_err(tx_err_to_app)
    }

    pub async fn create_budget(
        &self,
        schema: &str,
        user_id: Uuid,
        name: String,
        limit_idr: i64,
        category_id: Option<Uuid>,
        period_type: BudgetPeriodType,
        start_date: DateTime<Utc>,
        end_date: Option<DateTime<Utc>>,
        alert_threshold: i32,
        daily_limit_enabled: bool,
        _limits: &TierLimits,
    ) -> Result<BudgetWithProgress, AppError> {
        let now = Utc::now();
        let id = Uuid::new_v4();
        let budget = Budget {
            id,
            user_id,
            sync_id: id.to_string(),
            name,
            limit_idr,
            category_id,
            period_type,
            start_date,
            end_date,
            alert_threshold,
            daily_limit_enabled,
            is_deleted: false,
            deleted_at: None,
            created_at: now,
            updated_at: now,
        };
        self.budget_repo.create(schema, &budget).await.map_err(tx_err_to_app)?;

        let category_name = if let Some(cid) = category_id {
            self.cat_repo
                .get_by_id(schema, cid)
                .await
                .map_err(tx_err_to_app)?
                .map(|c| c.name)
                .unwrap_or_default()
        } else {
            String::new()
        };

        Ok(BudgetWithProgress { budget, spent_idr: 0, category_name })
    }

    pub async fn get_budget(
        &self,
        schema: &str,
        id: Uuid,
        user_id: Uuid,
    ) -> Result<BudgetWithProgress, AppError> {
        self.budget_repo
            .get_by_id(schema, id, user_id)
            .await
            .map_err(tx_err_to_app)?
            .ok_or(AppError::NotFound)
    }

    pub async fn update_budget(
        &self,
        schema: &str,
        id: Uuid,
        user_id: Uuid,
        name: Option<String>,
        limit_idr: Option<i64>,
        category_id: Option<Uuid>,
        period_type: Option<BudgetPeriodType>,
        start_date: Option<DateTime<Utc>>,
        end_date: Option<DateTime<Utc>>,
        alert_threshold: Option<i32>,
        daily_limit_enabled: Option<bool>,
    ) -> Result<BudgetWithProgress, AppError> {
        let bwp = self
            .budget_repo
            .get_by_id(schema, id, user_id)
            .await
            .map_err(tx_err_to_app)?
            .ok_or(AppError::NotFound)?;

        let mut b = bwp.budget;
        if let Some(v) = name { b.name = v; }
        if let Some(v) = limit_idr { b.limit_idr = v; }
        if let Some(v) = category_id { b.category_id = Some(v); }
        if let Some(v) = period_type { b.period_type = v; }
        if let Some(v) = start_date { b.start_date = v; }
        if let Some(v) = end_date { b.end_date = Some(v); }
        if let Some(v) = alert_threshold { b.alert_threshold = v; }
        if let Some(v) = daily_limit_enabled { b.daily_limit_enabled = v; }

        self.budget_repo.update(schema, &b).await.map_err(tx_err_to_app)?;

        self.budget_repo
            .get_by_id(schema, id, user_id)
            .await
            .map_err(tx_err_to_app)?
            .ok_or(AppError::NotFound)
    }

    pub async fn delete_budget(&self, schema: &str, id: Uuid, user_id: Uuid) -> Result<(), AppError> {
        self.budget_repo.soft_delete(schema, id, user_id).await.map_err(tx_err_to_app)
    }
}
