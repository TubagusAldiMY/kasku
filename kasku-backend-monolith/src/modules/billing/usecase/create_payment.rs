use std::sync::Arc;
use chrono::Utc;
use uuid::Uuid;

use crate::modules::billing::domain::{
    entity::Payment,
    error::BillingError,
    repository::{PaymentRepository, SubscriptionPlanRepository, SubscriptionRepository},
};

pub struct CreatePaymentInput {
    pub user_id: Uuid,
    pub plan_id: Uuid,
}

pub struct CreatePaymentOutput {
    pub order_id: String,
    pub amount_idr: i64,
    pub payment_url: Option<String>,
}

pub struct CreatePaymentUseCase {
    sub_repo: Arc<dyn SubscriptionRepository>,
    plan_repo: Arc<dyn SubscriptionPlanRepository>,
    payment_repo: Arc<dyn PaymentRepository>,
    orchestrator_url: Option<String>,
}

impl CreatePaymentUseCase {
    pub fn new(
        sub_repo: Arc<dyn SubscriptionRepository>,
        plan_repo: Arc<dyn SubscriptionPlanRepository>,
        payment_repo: Arc<dyn PaymentRepository>,
        orchestrator_url: Option<String>,
    ) -> Self {
        Self { sub_repo, plan_repo, payment_repo, orchestrator_url }
    }

    pub async fn execute(&self, input: CreatePaymentInput) -> Result<CreatePaymentOutput, BillingError> {
        let plan = self.plan_repo.find_by_id(input.plan_id).await?
            .ok_or(BillingError::PlanNotFound)?;

        if plan.price_idr == 0 {
            return Err(BillingError::Internal(anyhow::anyhow!("tidak bisa membayar plan gratis")));
        }

        let now = Utc::now();
        let order_id = format!("KSK-{}-{}", input.user_id.simple(), Uuid::new_v4().simple());

        let payment = Payment {
            id: Uuid::new_v4(),
            user_id: input.user_id,
            plan_id: input.plan_id,
            order_id: order_id.clone(),
            amount_idr: plan.price_idr as i64,
            status: "PENDING".to_string(),
            payment_method: None,
            duration_days: 30,
            orchestrator_ref: None,
            paid_at: None,
            expired_at: Some(now + chrono::Duration::hours(24)),
            created_at: now,
            updated_at: now,
        };

        self.payment_repo.create(&payment).await?;

        Ok(CreatePaymentOutput {
            order_id,
            amount_idr: payment.amount_idr,
            payment_url: None,
        })
    }
}
