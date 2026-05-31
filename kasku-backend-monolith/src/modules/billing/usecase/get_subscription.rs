use std::sync::Arc;
use uuid::Uuid;

use crate::modules::billing::domain::{
    entity::{Subscription, SubscriptionPlan},
    error::BillingError,
    repository::{SubscriptionPlanRepository, SubscriptionRepository},
};

pub struct SubscriptionDetail {
    pub subscription: Subscription,
    pub plan: SubscriptionPlan,
}

pub struct GetSubscriptionUseCase {
    sub_repo: Arc<dyn SubscriptionRepository>,
    plan_repo: Arc<dyn SubscriptionPlanRepository>,
}

impl GetSubscriptionUseCase {
    pub fn new(sub_repo: Arc<dyn SubscriptionRepository>, plan_repo: Arc<dyn SubscriptionPlanRepository>) -> Self {
        Self { sub_repo, plan_repo }
    }

    pub async fn execute(&self, user_id: Uuid) -> Result<SubscriptionDetail, BillingError> {
        let sub = self.sub_repo.find_by_user_id(user_id).await?
            .ok_or(BillingError::SubscriptionNotFound)?;

        let plan = self.plan_repo.find_by_id(sub.plan_id).await?
            .ok_or(BillingError::PlanNotFound)?;

        Ok(SubscriptionDetail { subscription: sub, plan })
    }
}
