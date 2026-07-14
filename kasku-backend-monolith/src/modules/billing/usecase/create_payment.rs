use std::sync::Arc;
use chrono::Utc;
use uuid::Uuid;

use crate::modules::billing::domain::{
    entity::Payment,
    error::BillingError,
    repository::{PaymentRepository, SubscriptionPlanRepository, SubscriptionRepository},
};
use crate::modules::billing::infrastructure::orchestrator_client::{DepositRequest, OrchestratorClient};

/// Metode pembayaran default saat request tidak menyebutkan (monolith belum expose
/// pilihan metode di API). QRIS dipilih sebagai default paling umum.
const DEFAULT_PAYMENT_METHOD: &str = "QRIS";

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
    orchestrator: Option<Arc<OrchestratorClient>>,
}

impl CreatePaymentUseCase {
    pub fn new(
        sub_repo: Arc<dyn SubscriptionRepository>,
        plan_repo: Arc<dyn SubscriptionPlanRepository>,
        payment_repo: Arc<dyn PaymentRepository>,
        orchestrator: Option<Arc<OrchestratorClient>>,
    ) -> Self {
        Self { sub_repo, plan_repo, payment_repo, orchestrator }
    }

    pub async fn execute(&self, input: CreatePaymentInput) -> Result<CreatePaymentOutput, BillingError> {
        let plan = self.plan_repo.find_by_id(input.plan_id).await?
            .ok_or(BillingError::PlanNotFound)?;

        if plan.price_idr == 0 {
            return Err(BillingError::Internal(anyhow::anyhow!("tidak bisa membayar plan gratis")));
        }

        let now = Utc::now();
        let amount_idr = plan.price_idr as i64;
        let duration_days = 30;
        let order_id = format!("KSK-{}-{}", input.user_id.simple(), Uuid::new_v4().simple());

        // Inisiasi ke Payment Orchestrator (jika dikonfigurasi). order_id dipakai sebagai
        // refId + Idempotency-Key. Payment record baru disimpan SETELAH orchestrator sukses,
        // meniru billing-service microservice (hindari PENDING lokal tanpa transaksi gateway).
        let (payment_url, orchestrator_ref, expired_at) = match &self.orchestrator {
            Some(client) => {
                let req = DepositRequest {
                    ref_id: order_id.clone(),
                    amount: amount_idr,
                    currency: "IDR".to_string(),
                    payment_method: DEFAULT_PAYMENT_METHOD.to_string(),
                    remarks: format!("Berlangganan KasKu {}", plan.name),
                };
                let data = client.initiate_deposit(&req, &order_id).await.map_err(|e| {
                    tracing::error!(error = %e, order_id = %order_id, "payment orchestrator gagal menginisiasi deposit");
                    BillingError::PaymentGatewayUnavailable
                })?;
                let url = if data.payment_url.is_empty() { None } else { Some(data.payment_url) };
                let oref = if data.provider_trx_id.is_empty() { None } else { Some(data.provider_trx_id) };
                let exp = data.expires_at.unwrap_or(now + chrono::Duration::hours(24));
                (url, oref, exp)
            }
            None => {
                // Dev / orchestrator belum dikonfigurasi — simpan PENDING lokal tanpa payment_url.
                tracing::warn!(order_id = %order_id, "orchestrator tidak dikonfigurasi — payment dibuat PENDING tanpa payment_url");
                (None, None, now + chrono::Duration::hours(24))
            }
        };

        let payment = Payment {
            id: Uuid::new_v4(),
            user_id: input.user_id,
            plan_id: input.plan_id,
            order_id: order_id.clone(),
            amount_idr,
            status: "PENDING".to_string(),
            payment_method: Some(DEFAULT_PAYMENT_METHOD.to_string()),
            duration_days,
            orchestrator_ref,
            paid_at: None,
            expired_at: Some(expired_at),
            created_at: now,
            updated_at: now,
        };

        self.payment_repo.create(&payment).await?;

        Ok(CreatePaymentOutput {
            order_id,
            amount_idr,
            payment_url,
        })
    }
}
