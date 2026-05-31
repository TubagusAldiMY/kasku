use std::sync::Arc;
use crate::modules::billing::domain::{entity::SubscriptionPlan, error::BillingError, repository::SubscriptionPlanRepository};

pub struct ListPlansUseCase {
    plan_repo: Arc<dyn SubscriptionPlanRepository>,
}

impl ListPlansUseCase {
    pub fn new(plan_repo: Arc<dyn SubscriptionPlanRepository>) -> Self {
        Self { plan_repo }
    }

    pub async fn execute(&self) -> Result<Vec<SubscriptionPlan>, BillingError> {
        self.plan_repo.list_active().await
    }
}
