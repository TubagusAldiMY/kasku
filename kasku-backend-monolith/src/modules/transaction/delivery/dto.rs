use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::modules::transaction::domain::entity::{
    Budget, BudgetWithProgress, Category, Transaction, TransactionSummary,
};

// ---------- Transaction DTOs ----------

#[derive(Debug, Serialize)]
pub struct TransactionResponse {
    pub id: String,
    pub sync_id: String,
    pub account_id: String,
    pub category_id: Option<String>,
    pub budget_id: Option<String>,
    pub transaction_type: String,
    pub amount_idr: i64,
    pub transaction_date: String,
    pub notes: String,
    pub to_account_id: Option<String>,
    pub created_at: String,
}

impl From<Transaction> for TransactionResponse {
    fn from(t: Transaction) -> Self {
        Self {
            id: t.id.to_string(),
            sync_id: t.sync_id,
            account_id: t.account_id.to_string(),
            category_id: t.category_id.map(|u| u.to_string()),
            budget_id: t.budget_id.map(|u| u.to_string()),
            transaction_type: t.transaction_type.to_string(),
            amount_idr: t.amount_idr,
            transaction_date: t.transaction_date.to_rfc3339(),
            notes: t.notes,
            to_account_id: t.to_account_id.map(|u| u.to_string()),
            created_at: t.created_at.to_rfc3339(),
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct CreateTransactionRequest {
    pub account_id: Uuid,
    pub category_id: Option<Uuid>,
    pub budget_id: Option<Uuid>,
    pub transaction_type: String,
    pub amount_idr: i64,
    pub transaction_date: String,
    pub notes: Option<String>,
    pub to_account_id: Option<Uuid>,
    pub sync_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateTransactionRequest {
    pub account_id: Option<Uuid>,
    pub category_id: Option<Uuid>,
    pub budget_id: Option<Uuid>,
    pub transaction_type: Option<String>,
    pub amount_idr: Option<i64>,
    pub transaction_date: Option<String>,
    pub notes: Option<String>,
    pub to_account_id: Option<Uuid>,
}

#[derive(Debug, Deserialize)]
pub struct ListTransactionsQuery {
    pub from: Option<String>,
    pub to: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct TransactionSummaryResponse {
    pub total_income: i64,
    pub total_expense: i64,
    pub net_amount: i64,
}

impl From<TransactionSummary> for TransactionSummaryResponse {
    fn from(s: TransactionSummary) -> Self {
        Self {
            total_income: s.total_income,
            total_expense: s.total_expense,
            net_amount: s.net_amount,
        }
    }
}

// ---------- Category DTOs ----------

#[derive(Debug, Serialize)]
pub struct CategoryResponse {
    pub id: String,
    pub name: String,
    pub icon: String,
    pub color: String,
    pub category_type: String,
    pub is_default: bool,
    pub created_at: String,
}

impl From<Category> for CategoryResponse {
    fn from(c: Category) -> Self {
        Self {
            id: c.id.to_string(),
            name: c.name,
            icon: c.icon,
            color: c.color,
            category_type: c.category_type.to_string(),
            is_default: c.is_default,
            created_at: c.created_at.to_rfc3339(),
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct CreateCategoryRequest {
    pub name: String,
    #[serde(default = "default_icon")]
    pub icon: String,
    #[serde(default = "default_color")]
    pub color: String,
    pub category_type: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdateCategoryRequest {
    pub name: Option<String>,
    pub icon: Option<String>,
    pub color: Option<String>,
    pub category_type: Option<String>,
}

fn default_icon() -> String { "tag".into() }
fn default_color() -> String { "#6366f1".into() }

// ---------- Budget DTOs ----------

#[derive(Debug, Serialize)]
pub struct BudgetResponse {
    pub id: String,
    pub sync_id: String,
    pub name: String,
    pub limit_idr: i64,
    pub spent_idr: i64,
    pub remaining_idr: i64,
    pub category_id: Option<String>,
    pub category_name: String,
    pub period_type: String,
    pub start_date: String,
    pub end_date: Option<String>,
    pub alert_threshold: i32,
    pub daily_limit_enabled: bool,
    pub created_at: String,
}

impl From<BudgetWithProgress> for BudgetResponse {
    fn from(bwp: BudgetWithProgress) -> Self {
        let b = bwp.budget;
        Self {
            id: b.id.to_string(),
            sync_id: b.sync_id,
            name: b.name,
            limit_idr: b.limit_idr,
            spent_idr: bwp.spent_idr,
            remaining_idr: b.limit_idr - bwp.spent_idr,
            category_id: b.category_id.map(|u| u.to_string()),
            category_name: bwp.category_name,
            period_type: b.period_type.to_string(),
            start_date: b.start_date.to_rfc3339(),
            end_date: b.end_date.map(|d| d.to_rfc3339()),
            alert_threshold: b.alert_threshold,
            daily_limit_enabled: b.daily_limit_enabled,
            created_at: b.created_at.to_rfc3339(),
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct CreateBudgetRequest {
    pub name: String,
    pub limit_idr: i64,
    pub category_id: Option<Uuid>,
    #[serde(default = "default_period_type")]
    pub period_type: String,
    pub start_date: String,
    pub end_date: Option<String>,
    #[serde(default = "default_alert_threshold")]
    pub alert_threshold: i32,
    #[serde(default)]
    pub daily_limit_enabled: bool,
}

#[derive(Debug, Deserialize)]
pub struct UpdateBudgetRequest {
    pub name: Option<String>,
    pub limit_idr: Option<i64>,
    pub category_id: Option<Uuid>,
    pub period_type: Option<String>,
    pub start_date: Option<String>,
    pub end_date: Option<String>,
    pub alert_threshold: Option<i32>,
    pub daily_limit_enabled: Option<bool>,
}

fn default_period_type() -> String { "MONTHLY".into() }
fn default_alert_threshold() -> i32 { 80 }
