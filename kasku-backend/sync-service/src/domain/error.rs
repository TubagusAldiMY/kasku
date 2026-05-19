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

    // Tenant schema ada tapi sync_log belum di-provision oleh finance-service.
    // Terjadi jika provision_tenant() belum selesai saat sync pertama dilakukan.
    #[error("tenant '{0}' belum di-provisioning, coba beberapa saat lagi")]
    TenantNotProvisioned(String),

    #[error("upstream service unavailable")]
    UpstreamUnavailable,

    #[error("upstream service timeout")]
    UpstreamTimeout,

    #[error("upstream response invalid")]
    UpstreamInvalidResponse,
}
