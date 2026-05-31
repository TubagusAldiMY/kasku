use thiserror::Error;

#[derive(Debug, Error)]
pub enum BillingError {
    #[error("subscription tidak ditemukan")]
    SubscriptionNotFound,

    #[error("plan tidak ditemukan")]
    PlanNotFound,

    #[error("payment tidak ditemukan")]
    PaymentNotFound,

    #[error("webhook signature tidak valid")]
    InvalidWebhookSignature,

    #[error("payment sudah diproses")]
    PaymentAlreadyProcessed,

    #[error("internal error: {0}")]
    Internal(#[from] anyhow::Error),
}

impl From<sqlx::Error> for BillingError {
    fn from(e: sqlx::Error) -> Self {
        BillingError::Internal(anyhow::anyhow!(e))
    }
}
