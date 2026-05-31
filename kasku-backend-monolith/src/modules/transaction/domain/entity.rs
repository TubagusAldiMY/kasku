use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::fmt;
use std::str::FromStr;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TransactionType {
    Income,
    Expense,
    Transfer,
}

impl fmt::Display for TransactionType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            TransactionType::Income => write!(f, "INCOME"),
            TransactionType::Expense => write!(f, "EXPENSE"),
            TransactionType::Transfer => write!(f, "TRANSFER"),
        }
    }
}

impl FromStr for TransactionType {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "INCOME" => Ok(TransactionType::Income),
            "EXPENSE" => Ok(TransactionType::Expense),
            "TRANSFER" => Ok(TransactionType::Transfer),
            other => Err(format!("unknown transaction type: {}", other)),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub id: Uuid,
    pub sync_id: String,
    pub account_id: Uuid,
    pub category_id: Option<Uuid>,
    pub budget_id: Option<Uuid>,
    pub transaction_type: TransactionType,
    pub amount_idr: i64,
    pub transaction_date: DateTime<Utc>,
    pub notes: String,
    pub to_account_id: Option<Uuid>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl<'r> sqlx::FromRow<'r, sqlx::postgres::PgRow> for Transaction {
    fn from_row(row: &'r sqlx::postgres::PgRow) -> Result<Self, sqlx::Error> {
        use sqlx::Row;
        let type_str: String = row.try_get("transaction_type")?;
        let transaction_type = type_str.parse::<TransactionType>().map_err(|e| {
            sqlx::Error::ColumnDecode {
                index: "transaction_type".to_string(),
                source: Box::new(std::io::Error::new(std::io::ErrorKind::InvalidData, e)),
            }
        })?;
        Ok(Transaction {
            id: row.try_get("id")?,
            sync_id: row.try_get("sync_id")?,
            account_id: row.try_get("account_id")?,
            category_id: row.try_get("category_id")?,
            budget_id: row.try_get("budget_id")?,
            transaction_type,
            amount_idr: row.try_get("amount_idr")?,
            transaction_date: row.try_get("transaction_date")?,
            notes: row.try_get("notes")?,
            to_account_id: row.try_get("to_account_id")?,
            is_deleted: row.try_get("is_deleted")?,
            deleted_at: row.try_get("deleted_at")?,
            created_at: row.try_get("created_at")?,
            updated_at: row.try_get("updated_at")?,
        })
    }
}

#[derive(Debug, Clone)]
pub struct TransactionSummary {
    pub total_income: i64,
    pub total_expense: i64,
    pub net_amount: i64,
}

impl<'r> sqlx::FromRow<'r, sqlx::postgres::PgRow> for TransactionSummary {
    fn from_row(row: &'r sqlx::postgres::PgRow) -> Result<Self, sqlx::Error> {
        use sqlx::Row;
        let total_income: i64 = row.try_get("total_income")?;
        let total_expense: i64 = row.try_get("total_expense")?;
        Ok(TransactionSummary {
            total_income,
            total_expense,
            net_amount: total_income - total_expense,
        })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum CategoryType {
    Income,
    Expense,
    Both,
}

impl fmt::Display for CategoryType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CategoryType::Income => write!(f, "INCOME"),
            CategoryType::Expense => write!(f, "EXPENSE"),
            CategoryType::Both => write!(f, "BOTH"),
        }
    }
}

impl FromStr for CategoryType {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "INCOME" => Ok(CategoryType::Income),
            "EXPENSE" => Ok(CategoryType::Expense),
            "BOTH" => Ok(CategoryType::Both),
            other => Err(format!("unknown category type: {}", other)),
        }
    }
}

#[derive(Debug, Clone)]
pub struct Category {
    pub id: Uuid,
    pub name: String,
    pub icon: String,
    pub color: String,
    pub category_type: CategoryType,
    pub is_default: bool,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl<'r> sqlx::FromRow<'r, sqlx::postgres::PgRow> for Category {
    fn from_row(row: &'r sqlx::postgres::PgRow) -> Result<Self, sqlx::Error> {
        use sqlx::Row;
        let type_str: String = row.try_get("category_type")?;
        let category_type = type_str.parse::<CategoryType>().map_err(|e| {
            sqlx::Error::ColumnDecode {
                index: "category_type".to_string(),
                source: Box::new(std::io::Error::new(std::io::ErrorKind::InvalidData, e)),
            }
        })?;
        Ok(Category {
            id: row.try_get("id")?,
            name: row.try_get("name")?,
            icon: row.try_get("icon")?,
            color: row.try_get("color")?,
            category_type,
            is_default: row.try_get("is_default")?,
            is_deleted: row.try_get("is_deleted")?,
            deleted_at: row.try_get("deleted_at")?,
            created_at: row.try_get("created_at")?,
            updated_at: row.try_get("updated_at")?,
        })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum BudgetPeriodType {
    Monthly,
    Weekly,
    Custom,
}

impl fmt::Display for BudgetPeriodType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            BudgetPeriodType::Monthly => write!(f, "MONTHLY"),
            BudgetPeriodType::Weekly => write!(f, "WEEKLY"),
            BudgetPeriodType::Custom => write!(f, "CUSTOM"),
        }
    }
}

impl FromStr for BudgetPeriodType {
    type Err = String;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "MONTHLY" => Ok(BudgetPeriodType::Monthly),
            "WEEKLY" => Ok(BudgetPeriodType::Weekly),
            "CUSTOM" => Ok(BudgetPeriodType::Custom),
            other => Err(format!("unknown budget period type: {}", other)),
        }
    }
}

#[derive(Debug, Clone)]
pub struct Budget {
    pub id: Uuid,
    pub user_id: Uuid,
    pub sync_id: String,
    pub name: String,
    pub limit_idr: i64,
    pub category_id: Option<Uuid>,
    pub period_type: BudgetPeriodType,
    pub start_date: DateTime<Utc>,
    pub end_date: Option<DateTime<Utc>>,
    pub alert_threshold: i32,
    pub daily_limit_enabled: bool,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct BudgetWithProgress {
    pub budget: Budget,
    pub spent_idr: i64,
    pub category_name: String,
}
