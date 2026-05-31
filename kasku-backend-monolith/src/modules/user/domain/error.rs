use thiserror::Error;

#[derive(Debug, Error)]
pub enum UserError {
    #[error("profil tidak ditemukan")]
    ProfileNotFound,

    #[error("provisioning gagal: {0}")]
    ProvisioningFailed(String),

    #[error("internal error: {0}")]
    Internal(#[from] anyhow::Error),
}

impl From<sqlx::Error> for UserError {
    fn from(e: sqlx::Error) -> Self {
        UserError::Internal(anyhow::anyhow!(e))
    }
}
