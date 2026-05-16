use thiserror::Error;

#[derive(Debug, Error)]
pub enum DomainError {
    #[error("tenant schema tidak valid: {0}")]
    InvalidTenantSchema(String),

    #[error("tenant schema tidak cocok dengan user: expected {expected}, got {actual}")]
    TenantMismatch { expected: String, actual: String },

    #[error("tipe entitas tidak didukung: {0}")]
    UnsupportedEntityType(String),

    #[error("header autentikasi tidak ditemukan")]
    Unauthorized,

    #[error("kesalahan database")]
    DatabaseError(#[from] sqlx::Error),

    #[error("upstream service unavailable")]
    UpstreamUnavailable,

    #[error("upstream service timeout")]
    UpstreamTimeout,

    #[error("upstream response invalid")]
    UpstreamInvalidResponse,
}
