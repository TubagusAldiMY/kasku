use std::sync::Arc;
use async_trait::async_trait;
use uuid::Uuid;

use crate::modules::billing::domain::repository::{SubscriptionPlanRepository, SubscriptionRepository};
use crate::shared::middleware::tier_inject::TierLimits;
use super::GetTierLimitsUseCase;

pub struct GetTierLimitsUseCaseImpl {
    sub_repo: Arc<dyn SubscriptionRepository>,
    plan_repo: Arc<dyn SubscriptionPlanRepository>,
}

impl GetTierLimitsUseCaseImpl {
    pub fn new(sub_repo: Arc<dyn SubscriptionRepository>, plan_repo: Arc<dyn SubscriptionPlanRepository>) -> Self {
        Self { sub_repo, plan_repo }
    }
}

#[async_trait]
impl GetTierLimitsUseCase for GetTierLimitsUseCaseImpl {
    async fn get_tier_limits(&self, user_id: Uuid) -> anyhow::Result<TierLimits> {
        let sub = self.sub_repo.find_by_user_id(user_id).await
            .map_err(|e| anyhow::anyhow!(e))?;

        let plan_id = match sub {
            Some(s) if s.status == "ACTIVE" => s.plan_id,
            _ => return Ok(TierLimits::free()),
        };

        let plan = self.plan_repo.find_by_id(plan_id).await
            .map_err(|e| anyhow::anyhow!(e))?;

        Ok(match plan {
            Some(p) => parse_limits(&p.limits),
            None => TierLimits::free(),
        })
    }

    async fn get_tier_name(&self, user_id: Uuid) -> anyhow::Result<String> {
        let sub = self.sub_repo.find_by_user_id(user_id).await
            .map_err(|e| anyhow::anyhow!(e))?;

        let plan_id = match sub {
            Some(s) if s.status == "ACTIVE" => s.plan_id,
            _ => return Ok("FREE".to_string()),
        };

        let plan = self.plan_repo.find_by_id(plan_id).await
            .map_err(|e| anyhow::anyhow!(e))?;

        Ok(plan.map(|p| p.name).unwrap_or_else(|| "FREE".to_string()))
    }
}

fn parse_limits(v: &serde_json::Value) -> TierLimits {
    TierLimits {
        max_transactions_per_month: v["MaxTransactionsPerMonth"].as_i64().unwrap_or(50),
        max_financial_accounts: v["MaxFinancialAccounts"].as_i64().unwrap_or(3),
        max_investment_instruments: v["MaxInvestmentInstruments"].as_i64().unwrap_or(0),
        history_retention_months: v["HistoryRetentionMonths"].as_i64().unwrap_or(3),
        email_notifications_enabled: v["EmailNotificationsEnabled"].as_bool().unwrap_or(false),
        export_csv_enabled: v["ExportCsvEnabled"].as_bool().unwrap_or(false),
    }
}
