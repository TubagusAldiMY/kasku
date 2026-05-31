use thiserror::Error;

#[derive(Debug, Error)]
pub enum TransactionError {
    #[error("transaksi tidak ditemukan")]
    NotFound,

    #[error("kategori tidak ditemukan")]
    CategoryNotFound,

    #[error("kategori memiliki transaksi aktif")]
    CategoryHasTransactions,

    #[error("kategori default tidak dapat dihapus")]
    DefaultCategoryCannotBeDeleted,

    #[error("batas transaksi bulanan tercapai")]
    TransactionLimitReached,

    #[error("ekspor CSV tidak tersedia")]
    ExportNotAllowed,

    #[error("input tidak valid: {0}")]
    InvalidInput(String),

    #[error("saldo tidak mencukupi")]
    InsufficientBalance,

    #[error("rekening tidak ditemukan")]
    AccountNotFound,

    #[error("anggaran tidak ditemukan")]
    BudgetNotFound,

    #[error("batas anggaran tercapai")]
    BudgetLimitReached,

    #[error("tenant belum diprovisioning")]
    TenantNotProvisioned,

    #[error("internal error: {0}")]
    Internal(String),
}

impl From<sqlx::Error> for TransactionError {
    fn from(e: sqlx::Error) -> Self {
        if let sqlx::Error::Database(ref db) = e {
            if db.code().as_deref() == Some("42P01") {
                return TransactionError::TenantNotProvisioned;
            }
        }
        TransactionError::Internal(e.to_string())
    }
}
