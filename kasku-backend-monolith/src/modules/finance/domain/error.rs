use thiserror::Error;

#[derive(Debug, Error)]
pub enum FinanceError {
    #[error("akun tidak ditemukan")]
    AccountNotFound,

    #[error("tenant belum diprovisioning")]
    TenantNotProvisioned,

    #[error("limit akun tercapai")]
    AccountLimitReached,

    #[error("internal error: {0}")]
    Internal(#[from] anyhow::Error),
}

impl From<sqlx::Error> for FinanceError {
    fn from(e: sqlx::Error) -> Self {
        match e {
            sqlx::Error::Database(ref db) if db.code().as_deref() == Some("42P01") => {
                FinanceError::TenantNotProvisioned
            }
            other => FinanceError::Internal(anyhow::anyhow!(other)),
        }
    }
}
