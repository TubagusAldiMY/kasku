use thiserror::Error;

#[derive(Debug, Error)]
pub enum InvestmentError {
    #[error("investment asset tidak ditemukan")]
    NotFound,

    #[error("tenant belum diprovisioning")]
    TenantNotProvisioned,

    #[error("internal error: {0}")]
    Internal(String),
}

impl From<sqlx::Error> for InvestmentError {
    fn from(e: sqlx::Error) -> Self {
        if let sqlx::Error::Database(ref db) = e {
            if db.code().as_deref() == Some("42P01") {
                return InvestmentError::TenantNotProvisioned;
            }
        }
        InvestmentError::Internal(e.to_string())
    }
}
