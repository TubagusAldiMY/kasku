pub mod cancel_subscription;
pub mod create_payment;
pub mod expire_subscriptions;
pub mod get_subscription;
pub mod get_tier_limits;
pub mod handle_webhook;
pub mod list_plans;

use std::sync::Arc;
use async_trait::async_trait;
use uuid::Uuid;

use crate::modules::billing::domain::repository::{PaymentRepository, SubscriptionPlanRepository, SubscriptionRepository};
use crate::shared::middleware::tier_inject::TierLimits;
use self::get_tier_limits::GetTierLimitsUseCaseImpl;

#[async_trait]
pub trait GetTierLimitsUseCase: Send + Sync {
    async fn get_tier_limits(&self, user_id: Uuid) -> anyhow::Result<TierLimits>;
    async fn get_tier_name(&self, user_id: Uuid) -> anyhow::Result<String>;
}

pub struct BillingUseCases {
    pub list_plans: list_plans::ListPlansUseCase,
    pub get_subscription: get_subscription::GetSubscriptionUseCase,
    pub create_payment: create_payment::CreatePaymentUseCase,
    pub cancel_subscription: cancel_subscription::CancelSubscriptionUseCase,
    pub expire_subscriptions: expire_subscriptions::ExpireSubscriptionsUseCase,
    pub get_tier_limits: Arc<GetTierLimitsUseCaseImpl>,
}

impl BillingUseCases {
    pub fn new(
        plan_repo: Arc<dyn SubscriptionPlanRepository>,
        sub_repo: Arc<dyn SubscriptionRepository>,
        payment_repo: Arc<dyn PaymentRepository>,
        orchestrator_url: Option<String>,
        webhook_secret: Option<String>,
        pool: sqlx_postgres::PgPool,
    ) -> Self {
        let tier_uc = Arc::new(GetTierLimitsUseCaseImpl::new(sub_repo.clone(), plan_repo.clone()));
        Self {
            list_plans: list_plans::ListPlansUseCase::new(plan_repo.clone()),
            get_subscription: get_subscription::GetSubscriptionUseCase::new(sub_repo.clone(), plan_repo.clone()),
            create_payment: create_payment::CreatePaymentUseCase::new(
                sub_repo.clone(), plan_repo.clone(), payment_repo.clone(), orchestrator_url,
            ),
            cancel_subscription: cancel_subscription::CancelSubscriptionUseCase::new(sub_repo.clone(), pool.clone()),
            expire_subscriptions: expire_subscriptions::ExpireSubscriptionsUseCase::new(sub_repo.clone(), pool.clone()),
            get_tier_limits: tier_uc,
        }
    }
}

#[async_trait]
impl GetTierLimitsUseCase for BillingUseCases {
    async fn get_tier_limits(&self, user_id: Uuid) -> anyhow::Result<TierLimits> {
        self.get_tier_limits.get_tier_limits(user_id).await
    }

    async fn get_tier_name(&self, user_id: Uuid) -> anyhow::Result<String> {
        self.get_tier_limits.get_tier_name(user_id).await
    }
}
