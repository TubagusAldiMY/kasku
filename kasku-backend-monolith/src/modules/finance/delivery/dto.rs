use serde::{Deserialize, Serialize};
use crate::modules::finance::domain::entity::{BalanceHistory, FinancialAccount};

#[derive(Debug, Serialize)]
pub struct AccountResponse {
    pub id: String,
    pub name: String,
    pub account_type: String,
    pub balance: i64,
    pub initial_balance: i64,
    pub currency: String,
    pub color: String,
    pub icon: String,
    pub is_default: bool,
    pub created_at: String,
}

impl From<FinancialAccount> for AccountResponse {
    fn from(a: FinancialAccount) -> Self {
        Self {
            id: a.id.to_string(),
            name: a.name,
            account_type: a.account_type,
            balance: a.balance,
            initial_balance: a.initial_balance,
            currency: a.currency,
            color: a.color,
            icon: a.icon,
            is_default: a.is_default,
            created_at: a.created_at.to_rfc3339(),
        }
    }
}

#[derive(Debug, Serialize)]
pub struct BalanceHistoryResponse {
    pub id: String,
    pub account_id: String,
    pub amount: i64,
    pub balance: i64,
    pub note: Option<String>,
    pub created_at: String,
}

impl From<BalanceHistory> for BalanceHistoryResponse {
    fn from(h: BalanceHistory) -> Self {
        Self {
            id: h.id.to_string(),
            account_id: h.account_id.to_string(),
            amount: h.amount,
            balance: h.balance,
            note: h.note,
            created_at: h.created_at.to_rfc3339(),
        }
    }
}

#[derive(Debug, Deserialize)]
pub struct CreateAccountRequest {
    pub name: String,
    #[serde(default = "default_account_type")]
    pub account_type: String,
    #[serde(default = "default_currency")]
    pub currency: String,
    #[serde(default = "default_color")]
    pub color: String,
    #[serde(default = "default_icon")]
    pub icon: String,
    #[serde(default)]
    pub initial_balance: i64,
    #[serde(default)]
    pub is_default: bool,
}

fn default_account_type() -> String { "BANK".into() }
fn default_currency() -> String { "IDR".into() }
fn default_color() -> String { "#6366f1".into() }
fn default_icon() -> String { "wallet".into() }

#[derive(Debug, Deserialize)]
pub struct UpdateAccountRequest {
    pub name: Option<String>,
    pub color: Option<String>,
    pub icon: Option<String>,
    pub is_default: Option<bool>,
}
